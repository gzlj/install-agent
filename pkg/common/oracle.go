package common

import (
	"database/sql"
	"fmt"
	"log"
	//_ "github.com/mattn/go-oci8"
	_ "gopkg.in/goracle.v2"
)

var (
	G_OracleDb Oracle
)

type Oracle struct {
	conn *sql.DB
}

func InitOracleDb(ip, port , dbName, user, passwd string) (err error){
	dataSource := fmt.Sprintf("%s/%s@%s:%s/%s", user, passwd, ip, port, dbName)
	//db, err := sql.Open("oci8", fmt.Sprintf("%s/%s@%s:%s/%s", user, passwd, ip, port, dbName))


	//db, err := sql.Open("goracle", "LY_MSSP/LY_MSSP@192.168.30.60:1521/orclpdb")
	db, err := sql.Open("goracle", dataSource)
	if err != nil {
		log.Fatal(err)
		return
	}
	G_OracleDb = Oracle {
		conn: db,
	}
	log.Printf("Connectd successfully.")
	return
}

func (o Oracle) sqlExec(sqlStmt string) (err error ){
	//var (
	//	r sql.Result
	//)
	_, err = o.conn.Exec(sqlStmt)
	if err != nil {
		log.Fatal(err)
		return err
	}
	return
}


func (o Oracle) UpdateJobStatus(jobId, phase string) (err error ){
	sql := "update LY_JQ_BS_RW set ZT = '" +  phase + "' WHERE RWBH ='" + jobId + "'"
	err = o.sqlExec(sql)
	return
}


