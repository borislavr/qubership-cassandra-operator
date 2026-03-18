package cassandra

import (
	"fmt"
	"time"

	"github.com/gocql/gocql"
)

type ClusterInterface interface {
	NewSession() (Session, error)
}

type Cluster struct {
	cluster *gocql.ClusterConfig
}

func (r *Cluster) NewSession() (Session, error) {
	var session *gocql.Session
	var err error

	for try := 0; try < 3; try++ {
		r.cluster.Timeout = time.Duration(int(r.cluster.Timeout.Seconds())*(try+1)) * time.Second
		//r.cluster.ConnectTimeout = time.Duration(int(r.cluster.ConnectTimeout.Seconds())*(try+1)) * time.Second
		session, err = r.cluster.CreateSession()
		if err == nil {
			break
		}
		time.Sleep(time.Duration(2*(try+1)) * time.Second)
	}

	return &SessionImpl{session}, err
}

type Session interface {
	Query(string, ...interface{}) QueryInterface
	SetConsistency(consistency gocql.Consistency)
	Close()
}

type QueryInterface interface {
	Exec(panicIfError bool) error
	Iter() IterInterface
}

type IterInterface interface {
	Scan(...interface{}) bool
	RowData() (RowData, error)
	Close() error
}

type SessionImpl struct {
	session *gocql.Session
}

func (s *SessionImpl) Query(stmt string, values ...interface{}) QueryInterface {
	return &Query{s.session.Query(stmt, values...)}
}

func (s *SessionImpl) SetConsistency(consistency gocql.Consistency) {
	s.session.SetConsistency(consistency)
}

func (s *SessionImpl) Close() {
	if s.session != nil {
		s.session.Close()
	}
}

type Query struct {
	query *gocql.Query
}

func (q *Query) Iter() IterInterface {
	return &Iter{q.query.Iter(), q.query.Statement()}
}
func (q *Query) Exec(panicOnError bool) error {
	if err := q.Iter().Close(); err != nil {
		if panicOnError {
			panic(fmt.Sprintf("Error during query execution %s: %v", q.query.Statement(), err))
		}
		return err
	}
	return nil
}

type RowData struct {
	RowData gocql.RowData
}

func (r *RowData) GetValue(column string) interface{} {
	for i, col := range r.RowData.Columns {
		if col == column {
			return r.RowData.Values[i]
		}
	}
	return nil
}

type Iter struct {
	iter      *gocql.Iter
	statement string
}

func (i *Iter) Scan(dest ...interface{}) bool {
	return i.iter.Scan(dest...)
}

func (i *Iter) RowData() (RowData, error) {
	result, err := i.iter.RowData()
	if err != nil {
		return RowData{}, fmt.Errorf("Error during getting row data for statement %s, err: %v", i.statement, err)
	}
	return RowData{RowData: result}, nil
}

func (i *Iter) Close() error {
	return i.iter.Close()
}
