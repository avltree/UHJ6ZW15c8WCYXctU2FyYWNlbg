package simplerepo

import (
	"../object"
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"time"
)

type SimpleRepo struct {
	Dsn string
}

func (repo *SimpleRepo) GetHistory(o *object.Object) ([]*object.ObjectHistory, error) {
	db := repo.getDbConnection()
	defer db.Close()

	results, err := db.Query("select created_at, duration, response from response where object_id = ?", o.Id)
	var ret []*object.ObjectHistory

	if nil != err {
		return nil, err
	}

	for results.Next() {
		var oh object.ObjectHistory
		var createdAt time.Time
		err = results.Scan(&createdAt, &oh.Duration, &oh.Response)

		if nil != err {
			return nil, err
		}

		oh.CreatedAt = createdAt.Unix()
		ret = append(ret, &oh)
	}

	return ret, nil
}

func (repo *SimpleRepo) AddResponse(o *object.Object, response string, duration time.Duration) error {
	db := repo.getDbConnection()
	defer db.Close()

	_, err := db.Query(
		"insert into response(object_id, duration, response) values (?, ?, ?)",
		o.Id,
		duration.Seconds(),
		response,
	)

	return err
}

func (repo *SimpleRepo) Save(o *object.Object) error {
	db := repo.getDbConnection()
	defer db.Close()

	// TODO handle errors properly
	stmt, err := db.Prepare("insert into objects(url, `interval`) values (?, ?)")
	result, err := stmt.Exec(o.Url, o.Interval)
	o.Id, err = result.LastInsertId()

	return err
}

func (repo *SimpleRepo) FindOne(id int64) (*object.Object, error) {
	db := repo.getDbConnection()
	defer db.Close()

	var o object.Object
	result, err := db.Query("select url, `interval` from objects where id = ?", id)

	if nil != err {
		return nil, err
	}

	// TODO this and the find all should use the same "hydration"
	success := result.Next()

	if !success {
		return nil, errors.New(fmt.Sprintf("object with id: %d not found", id))
	}

	err = result.Scan(&o.Url, &o.Interval)
	o.Id = id

	return &o, err
}

func (repo *SimpleRepo) FindAll() []*object.Object {
	db := repo.getDbConnection()
	defer db.Close()

	// TODO fix error handling
	results, err := db.Query("select o.id, o.url, o.`interval`, max(r.created_at) as last_check " +
		"from objects o " +
		"left join response r on o.id = r.object_id " +
		"group by 1, 2, 3")
	if nil != err {
		panic(err)
	}

	var ret []*object.Object

	for results.Next() {
		var o object.Object
		var t sql.NullTime
		// TODO handle error
		results.Scan(&o.Id, &o.Url, &o.Interval, &t)

		if t.Valid {
			o.LastCheck = t.Time
		}

		ret = append(ret, &o)
	}

	return ret
}

func (repo *SimpleRepo) Delete(o *object.Object) error {
	db := repo.getDbConnection()
	defer db.Close()

	_, err := db.Query("delete from objects where id = ?", o.Id)

	return err
}

func (repo *SimpleRepo) getDbConnection () *sql.DB {
	// TODO don't hardcode this
	db, err := sql.Open("mysql", repo.Dsn)
	//db, err := sql.Open("mysql", "root:root@tcp(mysql:3306)/object_storage?parseTime=true")

	if err != nil {
		panic(err)
	}

	return db
}
