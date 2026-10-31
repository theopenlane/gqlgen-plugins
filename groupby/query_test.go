package groupby

import (
	"testing"

	"entgo.io/contrib/entgql"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testKeyExpr = `COALESCE("vulnerabilities"."cve_id", '')`

func buildQuery(t *testing.T, modifier func(*sql.Selector)) (string, []any) {
	t.Helper()

	s := sql.Dialect(dialect.Postgres).Select().From(sql.Table("vulnerabilities"))
	modifier(s)

	return s.Query()
}

func TestNewQueryDefaults(t *testing.T) {
	q := NewQuery("cve_id", nil, nil, nil, new(10), nil, nil)

	assert.False(t, q.byKey)
	assert.Equal(t, entgql.OrderDirectionDesc, q.direction)
	assert.False(t, q.backward)
	assert.Equal(t, 10, q.limit)
}

func TestNewQueryOrderAndLast(t *testing.T) {
	q := NewQuery("cve_id", nil, &Order{Field: OrderFieldKey, Direction: entgql.OrderDirectionAsc}, nil, nil, nil, new(5))

	assert.True(t, q.byKey)
	assert.Equal(t, entgql.OrderDirectionAsc, q.direction)
	assert.True(t, q.backward)
	assert.Equal(t, 5, q.limit)
}

func TestModifierCountDescAfter(t *testing.T) {
	q := NewQuery("cve_id", nil, nil, &entgql.Cursor[string]{ID: "CVE-1", Value: 5}, new(10), nil, nil)

	query, args := buildQuery(t, q.Modifier)

	assert.Contains(t, query, testKeyExpr+` AS "group_key"`)
	assert.Contains(t, query, `COUNT(*) AS "group_count"`)
	assert.Contains(t, query, "GROUP BY "+testKeyExpr)
	assert.Contains(t, query, "(COUNT(*), "+testKeyExpr+") < ($1, $2)")
	assert.Contains(t, query, "ORDER BY COUNT(*) DESC, "+testKeyExpr+" DESC")
	assert.Contains(t, query, "LIMIT 11")
	assert.Equal(t, []any{5, "CVE-1"}, args)
}

func TestModifierCountDescBackward(t *testing.T) {
	q := NewQuery("cve_id", nil, nil, nil, nil, &entgql.Cursor[string]{ID: "CVE-9", Value: 2}, new(3))

	query, args := buildQuery(t, q.Modifier)

	assert.Contains(t, query, "(COUNT(*), "+testKeyExpr+") > ($1, $2)")
	assert.Contains(t, query, "ORDER BY COUNT(*) ASC, "+testKeyExpr+" ASC")
	assert.Contains(t, query, "LIMIT 4")
	assert.Equal(t, []any{2, "CVE-9"}, args)
}

func TestModifierCountCursorWithoutValue(t *testing.T) {
	q := NewQuery("cve_id", nil, nil, &entgql.Cursor[string]{ID: "CVE-1"}, new(10), nil, nil)

	query, args := buildQuery(t, q.Modifier)

	assert.Contains(t, query, testKeyExpr+" < $1")
	assert.Equal(t, []any{"CVE-1"}, args)
}

func TestModifierKeyAsc(t *testing.T) {
	q := NewQuery("cve_id", nil, &Order{Field: OrderFieldKey, Direction: entgql.OrderDirectionAsc}, &entgql.Cursor[string]{ID: "CVE-1", Value: 5}, new(10), nil, nil)

	query, args := buildQuery(t, q.Modifier)

	assert.Contains(t, query, testKeyExpr+" > $1")
	assert.Contains(t, query, "ORDER BY "+testKeyExpr+" ASC")
	assert.NotContains(t, query, "ORDER BY COUNT(*)")
	assert.Equal(t, []any{"CVE-1"}, args)
}

func TestModifierWithoutCursor(t *testing.T) {
	q := NewQuery("cve_id", nil, nil, nil, new(10), nil, nil)

	query, args := buildQuery(t, q.Modifier)

	assert.NotContains(t, query, "HAVING")
	assert.Empty(t, args)
}

func TestTotalModifier(t *testing.T) {
	q := NewQuery("cve_id", nil, nil, nil, new(10), nil, nil)

	query, _ := buildQuery(t, q.TotalModifier)

	assert.Contains(t, query, "SELECT COUNT(DISTINCT "+testKeyExpr+")")
	assert.NotContains(t, query, "GROUP BY")
}

func TestConnectionForward(t *testing.T) {
	q := NewQuery("cve_id", nil, nil, nil, new(2), nil, nil)

	conn := q.Connection([]Row{{Key: "CVE-1", Count: 9}, {Key: "", Count: 4}, {Key: "CVE-2", Count: 1}}, 3)

	require.Len(t, conn.Edges, 2)
	assert.Equal(t, "CVE-1", *conn.Edges[0].Node.Key)
	assert.Equal(t, 9, conn.Edges[0].Node.Count)
	assert.Nil(t, conn.Edges[1].Node.Key)
	assert.Equal(t, entgql.Cursor[string]{ID: "CVE-1", Value: 9}, conn.Edges[0].Cursor)
	assert.True(t, conn.PageInfo.HasNextPage)
	assert.False(t, conn.PageInfo.HasPreviousPage)
	assert.Equal(t, &conn.Edges[0].Cursor, conn.PageInfo.StartCursor)
	assert.Equal(t, &conn.Edges[1].Cursor, conn.PageInfo.EndCursor)
	assert.Equal(t, 3, conn.TotalCount)
}

func severityFilter(s *sql.Selector) {
	s.Where(sql.EQ(s.C("severity"), "critical"))
}

func TestConnectionBucketFilter(t *testing.T) {
	q := NewQuery("cve_id", severityFilter, nil, nil, new(10), nil, nil)

	conn := q.Connection([]Row{{Key: "CVE-1", Count: 9}, {Key: "", Count: 4}}, 0)
	require.Len(t, conn.Edges, 2)

	query, args := buildQuery(t, conn.Edges[0].Node.Filter)

	assert.Contains(t, query, `"vulnerabilities"."severity" = $1`)
	assert.Contains(t, query, `"vulnerabilities"."cve_id" = $2`)
	assert.NotContains(t, query, "COALESCE")
	assert.Equal(t, []any{"critical", "CVE-1"}, args)

	query, args = buildQuery(t, conn.Edges[1].Node.Filter)

	assert.Contains(t, query, testKeyExpr+" = $2")
	assert.Equal(t, []any{"critical", ""}, args)
}

func TestConnectionBucketFilterWithoutBase(t *testing.T) {
	q := NewQuery("cve_id", nil, nil, nil, new(10), nil, nil)

	conn := q.Connection([]Row{{Key: "CVE-1", Count: 9}}, 0)

	query, args := buildQuery(t, conn.Edges[0].Node.Filter)

	assert.Contains(t, query, `WHERE "vulnerabilities"."cve_id" = $1`)
	assert.Equal(t, []any{"CVE-1"}, args)
}

func TestConnectionBackward(t *testing.T) {
	q := NewQuery("cve_id", nil, nil, nil, nil, nil, new(2))

	conn := q.Connection([]Row{{Key: "CVE-3", Count: 1}, {Key: "CVE-2", Count: 2}, {Key: "CVE-1", Count: 3}}, 0)

	require.Len(t, conn.Edges, 2)
	assert.Equal(t, "CVE-2", *conn.Edges[0].Node.Key)
	assert.Equal(t, "CVE-3", *conn.Edges[1].Node.Key)
	assert.True(t, conn.PageInfo.HasPreviousPage)
	assert.False(t, conn.PageInfo.HasNextPage)
}

func TestConnectionEmpty(t *testing.T) {
	q := NewQuery("cve_id", nil, nil, nil, new(2), nil, nil)

	conn := q.Connection(nil, 0)

	assert.Empty(t, conn.Edges)
	assert.Nil(t, conn.PageInfo.StartCursor)
	assert.Nil(t, conn.PageInfo.EndCursor)
}

func TestAnnotationDecode(t *testing.T) {
	a := &Annotation{}

	require.NoError(t, a.Decode(map[string]any{"groupable": true}))
	assert.True(t, a.Groupable)
	assert.Equal(t, AnnotationName, Groupable().Name())
}
