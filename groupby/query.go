package groupby

import (
	"errors"
	"slices"

	"entgo.io/contrib/entgql"
	"entgo.io/ent/dialect/sql"
	"github.com/samber/lo"
)

const (
	keyColumn   = "group_key"
	countColumn = "group_count"
	countExpr   = "COUNT(*)"
)

// ErrInvalidField is returned when the requested group by field is not groupable
var ErrInvalidField = errors.New("invalid group by field")

// OrderField is the field group by buckets are ordered by, bound to the GraphQL enum with @goEnum
type OrderField string

const (
	// OrderFieldCount orders buckets by the number of rows in each
	OrderFieldCount OrderField = "COUNT"
	// OrderFieldKey orders buckets by the grouped value
	OrderFieldKey OrderField = "KEY"
)

// Order is the ordering of group by buckets
type Order struct {
	// Direction is the ordering direction
	Direction entgql.OrderDirection `json:"direction"`
	// Field is the field to order by
	Field OrderField `json:"field"`
}

// Bucket is a group of rows sharing the same value for the grouped field
type Bucket struct {
	// Key is the grouped value, nil when the rows have no value
	Key *string `json:"key,omitempty"`
	// Count is the number of rows in the group
	Count int `json:"count"`
	// Filter selects the rows in the group, used by the bucket's nested connection resolver
	Filter func(*sql.Selector) `json:"-"`
}

// bucketFilter matches the rows of one bucket within the group by query's filter
type bucketFilter struct {
	base   func(*sql.Selector)
	column string
	key    string
}

// Apply adds the bucket's predicates to the selector, matching empty keys the same way the group by folds them
func (f bucketFilter) Apply(s *sql.Selector) {
	if f.base != nil {
		f.base(s)
	}

	if f.key == "" {
		s.Where(sql.EQ(keyExpr(s, f.column), ""))

		return
	}

	s.Where(sql.EQ(s.C(f.column), f.key))
}

// Edge is a bucket with its pagination cursor
type Edge struct {
	// Node is the bucket
	Node *Bucket `json:"node"`
	// Cursor is the cursor of the bucket
	Cursor entgql.Cursor[string] `json:"cursor"`
}

// Connection is a paginated list of group by buckets
type Connection struct {
	// Edges are the buckets on the page
	Edges []*Edge `json:"edges"`
	// PageInfo is the pagination info for the page
	PageInfo entgql.PageInfo[string] `json:"pageInfo"`
	// TotalCount is the number of buckets across all pages
	TotalCount int `json:"totalCount"`
}

// Row is a scanned group by result row
type Row struct {
	// Key is the grouped value, empty when the rows have no value
	Key string `json:"group_key"`
	// Count is the number of rows in the group
	Count int `json:"group_count"`
}

// Query builds a paginated group by on a single column
type Query struct {
	column    string
	filter    func(*sql.Selector)
	byKey     bool
	direction entgql.OrderDirection
	after     *entgql.Cursor[string]
	before    *entgql.Cursor[string]
	limit     int
	backward  bool
}

// NewQuery returns a group by query for the column, defaulting to ordering by count descending
// filter is the query's where predicate, carried onto each bucket so its rows can be listed
func NewQuery(column string, filter func(*sql.Selector), order *Order, after *entgql.Cursor[string], first *int, before *entgql.Cursor[string], last *int) *Query {
	q := &Query{
		column:    column,
		filter:    filter,
		direction: entgql.OrderDirectionDesc,
		after:     after,
		before:    before,
	}

	if order != nil {
		q.byKey = order.Field == OrderFieldKey
		q.direction = order.Direction
	}

	switch {
	case first != nil:
		q.limit = *first
	case last != nil:
		q.limit = *last
		q.backward = true
	}

	return q
}

// keyExpr returns the grouped column with null and empty values folded together
func keyExpr(s *sql.Selector, column string) string {
	return "COALESCE(" + s.C(column) + ", '')"
}

// Modifier selects one row per distinct value with its count, applying ordering, cursors, and the page limit
func (q *Query) Modifier(s *sql.Selector) {
	key := keyExpr(s, q.column)

	s.Select(sql.As(key, keyColumn), sql.As(countExpr, countColumn)).
		GroupBy(key).
		ClearOrder()

	var predicates []*sql.Predicate

	if q.after != nil {
		predicates = append(predicates, q.cursorPredicate(key, q.after, q.direction))
	}

	if q.before != nil {
		predicates = append(predicates, q.cursorPredicate(key, q.before, q.direction.Reverse()))
	}

	if len(predicates) > 0 {
		s.Having(sql.And(predicates...))
	}

	direction := q.direction
	if q.backward {
		direction = direction.Reverse()
	}

	order := sql.Asc
	if direction == entgql.OrderDirectionDesc {
		order = sql.Desc
	}

	if q.byKey {
		s.OrderBy(order(key))
	} else {
		s.OrderBy(order(countExpr), order(key))
	}

	if q.limit > 0 {
		s.Limit(q.limit + 1)
	}
}

// TotalModifier selects the number of distinct values across all pages
func (q *Query) TotalModifier(s *sql.Selector) {
	s.Select(sql.Count(sql.Distinct(keyExpr(s, q.column)))).ClearOrder()
}

// cursorPredicate matches buckets past the cursor in the given direction, comparing only the key when the cursor has no count
func (q *Query) cursorPredicate(key string, c *entgql.Cursor[string], direction entgql.OrderDirection) *sql.Predicate {
	asc := direction == entgql.OrderDirectionAsc

	if q.byKey || c.Value == nil {
		if asc {
			return sql.GT(key, c.ID)
		}

		return sql.LT(key, c.ID)
	}

	if asc {
		return sql.CompositeGT([]string{countExpr, key}, c.Value, c.ID)
	}

	return sql.CompositeLT([]string{countExpr, key}, c.Value, c.ID)
}

// Connection builds the connection for the scanned rows of one page
func (q *Query) Connection(rows []Row, total int) *Connection {
	hasMore := q.limit > 0 && len(rows) > q.limit
	if hasMore {
		rows = rows[:q.limit]
	}

	if q.backward {
		slices.Reverse(rows)
	}

	conn := &Connection{
		Edges:      make([]*Edge, 0, len(rows)),
		TotalCount: total,
	}

	if q.backward {
		conn.PageInfo.HasPreviousPage = hasMore
	} else {
		conn.PageInfo.HasNextPage = hasMore
	}

	for _, row := range rows {
		conn.Edges = append(conn.Edges, &Edge{
			Node: &Bucket{
				Key:    lo.EmptyableToPtr(row.Key),
				Count:  row.Count,
				Filter: bucketFilter{base: q.filter, column: q.column, key: row.Key}.Apply,
			},
			Cursor: entgql.Cursor[string]{ID: row.Key, Value: row.Count},
		})
	}

	if len(conn.Edges) > 0 {
		conn.PageInfo.StartCursor = &conn.Edges[0].Cursor
		conn.PageInfo.EndCursor = &conn.Edges[len(conn.Edges)-1].Cursor
	}

	return conn
}
