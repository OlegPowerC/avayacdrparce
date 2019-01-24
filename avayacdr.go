package main

import (
	"fmt"
	"net"
	"bufio"
	"time"
	"strings"
	"strconv"
	_ "github.com/go-sql-driver/mysql"
	"database/sql"
	"os"
	"os/signal"
	"io/ioutil"
	"encoding/json"
	"os/exec"
	"net/url"
	"net/http"
	"crypto/tls"
	"regexp"
)

const longForm = "010206 1504 MST"
const AvayaMsgLen = 93

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

func sendsms(number string, extension string, calltime string, companyname string) {
	if len(number) > 10{
		if ldebuggmode > 0{
			fmt.Println("SENDSMS function: number is " + number)
		}
		trx := number[len(number)-10:]
		if ldebuggmode > 0{
			fmt.Println("Cutednumberis " + trx)
		}
		var pNum, pName string
		SelectString := fmt.Sprintf("SELECT phone,name from smsto WHERE phone=\"7%s\" AND sendsms=1;", trx)
		Qres := db.QueryRow(SelectString)
		if ldebuggmode > 0{
			fmt.Println("SQL string is " + SelectString)
		}
		Errs := Qres.Scan(&pNum, &pName)
		if Errs == sql.ErrNoRows {
			if ldebuggmode > 0{
				fmt.Println("No data selected")
			}
			return
		} else if Errs != nil {
			if ldebuggmode > 0{
				fmt.Println("Error")
			}
			return
		}
		SMSText := fmt.Sprintf("Уважаемый(ая) %s, в %s вам поступил вызов от абонента %s, с внутренним номером %s", pName, calltime, companyname, extension)
		if ldebuggmode > 0{
			fmt.Println(SMSText)
			fmt.Println(pNum,pName)
		}
		a31 := url.Values{}
		a31.Set("phone",pNum)
		a31.Set("mpname",SMSText)
		tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
		client := &http.Client{Transport: tr}
		reqpr, _ := http.NewRequest("POST",SMSurl,strings.NewReader(a31.Encode()))
		reqpr.Header.Add("Content-Type","application/x-www-form-urlencoded")
		_, errpr := client.Do(reqpr)
		if errpr != nil{
			if ldebuggmode > 0{
				fmt.Println("DoRequest err")
			}
		}else{
			fmt.Println("Посылка SMS уведамления для ",pName,"на номер",pNum)
		}
	}
}

func IsNumber (teststring string) bool{
	re :=regexp.MustCompile(`^([0-9]+){3,}#?$`)
	return re.Match([]byte(strings.TrimSpace(teststring)))
}

func handle(conn net.Conn) error {

	defer func() {
		fmt.Println("Закрыто соединение:", conn.RemoteAddr())
		conn.Close()
	}()

	r := bufio.NewReader(conn)
	w := bufio.NewWriter(conn)
	scanr := bufio.NewScanner(r)

	for {
		scanned := scanr.Scan()
		if !scanned {
			if err := scanr.Err(); err != nil {
				fmt.Println(err, conn.RemoteAddr())
				return err
			}
			break
		}

		vtems := scanr.Text()

		var vtemp string
		if len(vtems) >= AvayaMsgLen {
			//if  IsNumber(vtemp[recoffset.calling_number_start:recoffset.calling_number_end]) && IsNumber(vtemp[recoffset.called_number_start:recoffset.called_number_end]) {
				if len(vtems) > AvayaMsgLen {
					vtemp = vtems[len(vtems)-AvayaMsgLen:]
				} else {
					vtemp = vtems
				}
			if  IsNumber(vtemp[recoffset.calling_number_start:recoffset.calling_number_end]) && IsNumber(vtemp[recoffset.called_number_start:recoffset.called_number_end]) {
				if ldebuggmode > 0 {
					fmt.Println(vtems)
				}
				var fr1 CDR_Record_1

				strdatef := vtemp[recoffset.dtime_start:recoffset.dtime_end] + " MSK"

				datetimed, _ := time.Parse(longForm, strdatef)
				fr1.dtime = datetimed
				fr1.duration, _ = strconv.Atoi(strings.TrimSpace(vtemp[recoffset.duration_start:recoffset.duration_end]))
				fr1.calling_number = strings.TrimSpace(vtemp[recoffset.calling_number_start:recoffset.calling_number_end])
				fr1.called_number = strings.TrimSpace(vtemp[recoffset.called_number_start:recoffset.called_number_end])
				udt := fr1.dtime.Unix()
				sut := strconv.FormatInt(udt, 10)
				qstr := fmt.Sprintf("INSERT INTO powerccdr(tm,duration,called,calling) VALUES (FROM_UNIXTIME(%s),%s,\"%s\",\"%s\")", sut, strconv.Itoa(fr1.duration), fr1.called_number, fr1.calling_number)
				//insert, err := db.Query(qstr)
				_, err := db.Exec(qstr)
				if err != nil {
					fmt.Println("Ошибка запроса INSERT:", err)
					panic(err.Error())
				}
				DateStr := fmt.Sprintf("%s-%s-%s %sч. %sмин.", vtemp[recoffset.dtime_start:recoffset.dtime_start+2],
					vtemp[recoffset.dtime_start+2:recoffset.dtime_start+4],
					vtemp[recoffset.dtime_start+4:recoffset.dtime_start+6],
					vtemp[recoffset.dtime_start+7:recoffset.dtime_start+9],
					vtemp[recoffset.dtime_start+9:recoffset.dtime_start+11])
				go sendsms(fr1.called_number, fr1.calling_number, DateStr, comname)
			}
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
	fmt.Println("Запущен сервис на:", addr)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Println("Ошибка:",err)
		return err
	}
	defer listener.Close()
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Ошибка подключения:", err)
			continue
		}
		fmt.Println("PowerCCDR подключение:", conn.RemoteAddr())
		handle(conn)
	}
}

//Описание JSON параметров для работы с базой данных
type params struct {
	Databasename string `json:"dbname"`
	Databaseuser string `json:"dbuser"`
	Callstable string `json:"calltable"`
	Databaseurl string `json:"dburl"`
	Debuggmode int `json:"debugmode"`
	Sendsmsurl string `json:"smsurl"`
	Companyname string `json:"company"`
}

var recoffset CDR_Record_offsett
var db *sql.DB
var errsql error
var ldebuggmode int
var comname string
var SMSurl string

const JsonFileName = "params.json"
func main() {
	var JParams params
	// Открываем файл с настройками
	jSettingsFile, err := os.Open(JsonFileName)
	// Проверяем на ошибки
	if err != nil {
		fmt.Println("Ошибка:",err)
	}
	defer jSettingsFile.Close()

	FData, err := ioutil.ReadAll(jSettingsFile)
	if err != nil {
		fmt.Println("Ошибка:",err)
	}

	json.Unmarshal(FData,&JParams)
	ldebuggmode = JParams.Debuggmode
	comname = JParams.Companyname
	SMSurl = JParams.Sendsmsurl

	fmt.Println("Режим отладки:",strconv.Itoa(JParams.Debuggmode))

	//Получение пароля из KeyRing посредством запуска Python скрипта
	databasepasswordby,err := exec.Command("./getuser.py","-u"+JParams.Databaseuser,"-k"+JParams.Databasename).Output()
	if err != nil{
		fmt.Println("Ошибка получения пароля для пользователя "+JParams.Databaseuser+" из KeyRing")
		os.Exit(1)
	}
	databasepassword := strings.TrimSpace(string(databasepasswordby))

	if databasepassword == "" {
		fmt.Println("Ошибка получения пароля для пользователя "+JParams.Databaseuser+" из KeyRing "+JParams.Databasename)
		os.Exit(1)
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs)
	//Афтономная функция для обработки сигналов ОС
	go func() {
		s := <- sigs
		fmt.Println("Принят сигнал ОС:",s)
		db.Close();
		os.Exit(1)
	}()

	//Создаем строку для соединения с базой данных
	DsToLog := fmt.Sprintf("%s@tcp(%s)/%s",JParams.Databaseuser,JParams.Databaseurl,JParams.Databasename)
	DsStr := fmt.Sprintf("%s:%s@tcp(%s)/%s",JParams.Databaseuser,databasepassword,JParams.Databaseurl,JParams.Databasename)
	fmt.Println("Попытка соединения с базой данных:", DsToLog)

	db,errsql = sql.Open("mysql",DsStr)
	if errsql != nil {
		fmt.Println("Ошибка:",errsql)
		panic(errsql.Error())
	}
	defer db.Close()

	err = db.Ping()
	if err != nil{
		fmt.Println("Ошибка:",err)
		panic(errsql.Error())
	}
	recoffset = CDR_Record_offsett{0,11,12,17,18,33,34,57}
	fmt.Println("Старт CDR сервиса")
	s1 := Server{":5001"}
	s1.ListenAndServe()
}
