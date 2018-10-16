package main

import (
	"fmt"
	"log"
	"net"
	"bufio"
	"time"
	"strings"
	"strconv"
_ "github.com/go-sql-driver/mysql"

	"database/sql"
	"flag"
)

const longForm = "010206 1504 MST"

type Server struct {
	Addr string
}

type Conn struct {
	net.Conn
	IdleTimeout time.Duration
}

func (c *Conn) Write(p []byte) (int, error) {
	c.updateDeadline()
	return c.Conn.Write(p)
}

func (c *Conn) Read(b []byte) (int, error) {
	c.updateDeadline()
	return c.Conn.Read(b)
}

func (c *Conn) updateDeadline() {
	idleDeadline := time.Now().Add(c.IdleTimeout)
	c.Conn.SetDeadline(idleDeadline)
}

type CDR_Record_1 struct{
	dtime time.Time
	calling_number string
	called_number string
	duration int
}

type CDR_Record_offsett struct{
dtime_start int
dtime_end int
duration_start int
duration_end int
calling_number_start int
calling_number_end int
called_number_start int
called_number_end int
}

func handle(conn net.Conn) error {

	defer func() {
		log.Printf("closing connection from %v", conn.RemoteAddr())
		conn.Close()
	}()
	r := bufio.NewReader(conn)
	w := bufio.NewWriter(conn)
	scanr := bufio.NewScanner(r)

	for {
		scanned := scanr.Scan()
		if !scanned {
			if err := scanr.Err(); err != nil {
				log.Printf("%v(%v)", err, conn.RemoteAddr())
				return err
			}
			break
		}

		vtems := scanr.Text()

		var vtemp string
		if len(vtems) >= 92{
			if len(vtems) > 92{
				vtemp = vtems[len(vtems)-92:]
			}else{
				vtemp = vtems
			}
			if *debuggmode == "1"{
				fmt.Println(vtems)
			}
			var fr1 CDR_Record_1

			strdatef := vtemp[recoffset.dtime_start:recoffset.dtime_end]+" MSK"

			datetimed,_ := time.Parse(longForm,strdatef)
			fr1.dtime = datetimed
			fr1.duration,_ = strconv.Atoi(strings.TrimSpace(vtemp[recoffset.duration_start:recoffset.duration_end]))
			fr1.calling_number = strings.TrimSpace(vtemp[recoffset.calling_number_start:recoffset.calling_number_end])
			fr1.called_number = strings.TrimSpace(vtemp[recoffset.called_number_start:recoffset.called_number_end])
			udt := fr1.dtime.Unix()
			sut := strconv.FormatInt(udt,10)
			qstr := "INSERT INTO powerccdr(tm,duration,called,calling) VALUES (FROM_UNIXTIME("+ sut +"),"+ strconv.Itoa(fr1.duration)+",\""+fr1.calling_number+"\",\""+fr1.called_number+"\")"

			insert, err := db.Query(qstr)


			if err != nil {
				panic(err.Error())
			}

			defer insert.Close()

		}
		w.Flush()
	}
	return nil
}

func (srv Server) ListenAndServe() error{
	addr := srv.Addr
	if addr == ""{
		addr = ":5001"
	}
	log.Printf("starting server on %v\n", addr)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Println(err)
		return err
	}
	defer listener.Close()
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("error accepting connection %v", err)
			continue
		}
		log.Printf("accepted connection from %v", conn.RemoteAddr())
		handle(conn) //TODO: Implement me
	}
}
var recoffset CDR_Record_offsett
var debuggmode * string
var db *sql.DB
var errsql error
func main() {
	debuggmode = flag.String("d","0","1 for debugg")
	sqluser := flag.String("u","powerccdr","username for SQL")
	sqlpassword := flag.String("p","test","password for SQL")
	sqldbname := flag.String("n","powerccdr","Database name")
	flag.Parse()

	db,errsql = sql.Open("mysql",*sqluser+":"+*sqlpassword+"@tcp(127.0.0.1:3306)/"+*sqldbname)
	if errsql != nil {
		panic(errsql.Error())
	}
	defer db.Close()
	recoffset = CDR_Record_offsett{0,11,12,16,17,32,33,56}
	fmt.Println("STARTED CDR Server")
	s1 := Server{":5001"}
	s1.ListenAndServe()
}
