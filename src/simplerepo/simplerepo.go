package simplerepo

import (
	"../object"
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"time"
)

// Implementation data structure, implements the repository interface from the object package
// FIXME Make the repo capable of handling multiple requests per DB connection
type SimpleRepo struct {
	Dsn string
}

// Gets the object response history from MySQL database
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

// Stores a new response for the specified object
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

// Saves an object into the database
// Note that only "insert" operations are permitted because of my interpretation of the POST resource
func (repo *SimpleRepo) Save(o *object.Object) error {
	db := repo.getDbConnection()
	defer db.Close()

	// TODO handle errors properly
	stmt, err := db.Prepare("insert into objects(url, `interval`) values (?, ?)")
	result, err := stmt.Exec(o.Url, o.Interval)
	o.Id, err = result.LastInsertId()

	return err
}

// Searches the database for a single object by its id
func (repo *SimpleRepo) FindOne(id int64) (*object.Object, error) {
	db := repo.getDbConnection()
	defer db.Close()

	var o object.Object
	// FIXME not important for the app's logic, but it doesn't retrieve and "hydrate" the LastCheck property
	result, err := db.Query("select url, `interval` from objects where id = ?", id)

	if nil != err {
		return nil, err
	}

	success := result.Next()

	if !success {
		return nil, errors.New(fmt.Sprintf("object with id: %d not found", id))
	}

	err = result.Scan(&o.Url, &o.Interval)
	o.Id = id

	return &o, err
}

// Returns all objects stored in the database
func (repo *SimpleRepo) FindAll() []*object.Object {
	db := repo.getDbConnection()
	defer db.Close()

	// TODO fix error handling
	// Joins to the "response" table so the latest response retrieval date is available
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

// Deletes object from the database
func (repo *SimpleRepo) Delete(o *object.Object) error {
	db := repo.getDbConnection()
	defer db.Close()

	_, err := db.Query("delete from objects where id = ?", o.Id)

	return err
}

// Helper function to initialize and get the DB instance.
func (repo *SimpleRepo) getDbConnection () *sql.DB {
	db, err := sql.Open("mysql", repo.Dsn)

	if err != nil {
		panic(err)
	}

	return db
}
