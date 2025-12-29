package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// В Go 01 - месяц с нулем впереди если одна цифра, то же самое 2 только дата а 06 год 2 цифры
const AvayaDateFormat = "010206 1504"
const AvayaMsgLen = 95

type Server struct {
	Addr string
	wg   *sync.WaitGroup
	quit chan struct{}
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

type CDR_Record_1 struct {
	dtime          time.Time
	originaldt     string
	calling_number string
	called_number  string
	duration       int
	condition_code string
}

type CDR_Record_offsett struct {
	dtime_start          int
	dtime_end            int
	duration_start       int
	duration_end         int
	calling_number_start int
	calling_number_end   int
	called_number_start  int
	called_number_end    int
	condition_code_start int
	condition_code_end   int
}

func NewServer(addr string) *Server {
	return &Server{
		Addr: addr,
		wg:   &sync.WaitGroup{},
		quit: make(chan struct{}),
	}
}

func sendsms(number string, extension string, calltime string, companyname string) {
	if len(number) > 10 {
		if ldebuggmode > 0 {
			fmt.Println("SENDSMS function: number is " + number)
		}
		trx := number[len(number)-10:]
		if ldebuggmode > 0 {
			fmt.Println("Cutednumberis " + trx)
		}
		var pNum, pName string

		Errs := db.QueryRow("SELECT phone,name from smsto WHERE phone=? AND sendsms=1;", "7"+trx).Scan(&pNum, &pName)

		if Errs == sql.ErrNoRows {
			if ldebuggmode > 0 {
				fmt.Println("No data selected")
			}
			return
		} else if Errs != nil {
			if ldebuggmode > 0 {
				fmt.Println("Error")
			}
			return
		}
		SMSText := fmt.Sprintf("Уважаемый(ая) %s, в %s вам поступил вызов от абонента %s, с внутренним номером %s", pName, calltime, companyname, extension)
		if ldebuggmode > 0 {
			fmt.Println(SMSText)
			fmt.Println(pNum, pName)
		}
		a31 := url.Values{}
		a31.Set("phone", pNum)
		a31.Set("mpname", SMSText)
		tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
		client := &http.Client{Transport: tr, Timeout: 10 * time.Second}
		reqpr, _ := http.NewRequest("POST", SMSurl, strings.NewReader(a31.Encode()))
		reqpr.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		_, errpr := client.Do(reqpr)
		if errpr != nil {
			if ldebuggmode > 0 {
				fmt.Println("DoRequest err")
			}
		} else {
			fmt.Println("Посылка SMS уведамления для ", pName, "на номер", pNum)
		}
	}
}

func IsNumber(teststring string) bool {
	re := regexp.MustCompile(`^([0-9]+){3,}#?$`)
	return re.Match([]byte(strings.TrimSpace(teststring)))
}

func (srv *Server) handle(ctx context.Context, conn net.Conn) {
	defer func() {
		srv.wg.Done()
		fmt.Println("Закрыто соединение:", conn.RemoteAddr())
		conn.Close()
	}()

	Timelocation, Ltimeloc := time.LoadLocation(Timezone)
	if Ltimeloc != nil {
		fmt.Println(Ltimeloc)
		Timelocation = time.UTC
	}

	r := bufio.NewReader(conn)
	w := bufio.NewWriter(conn)
	scanr := bufio.NewScanner(r)

	stmt, err := db.Prepare(`
					INSERT INTO powerccdr(tm, originaldt, duration, called, calling, cond) 
					VALUES (FROM_UNIXTIME(?), ?, ?, ?, ?, ?)
				`)
	if err != nil {
		fmt.Println("Ошибка подготовки запроса:", err)
		return
	}
	defer stmt.Close()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			scanned := scanr.Scan()
			if !scanned {
				if err := scanr.Err(); err != nil {
					fmt.Println(err, conn.RemoteAddr())
					return
				}
				return
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
				if IsNumber(strings.TrimSpace(vtemp[recoffset.calling_number_start:recoffset.calling_number_end])) && IsNumber(strings.TrimSpace(vtemp[recoffset.called_number_start:recoffset.called_number_end])) {
					if ldebuggmode > 0 {
						fmt.Println(vtems)
					}
					var fr1 CDR_Record_1

					strdatef := vtemp[recoffset.dtime_start:recoffset.dtime_end]

					datetimed, parseerr := time.ParseInLocation(AvayaDateFormat, strdatef, Timelocation)
					if parseerr != nil {
						fmt.Println(parseerr)
						fr1.dtime = time.Now().In(Timelocation)
					} else {
						fr1.dtime = datetimed
					}

					fr1.originaldt = vtemp[recoffset.dtime_start:recoffset.dtime_end]
					fr1.duration, _ = strconv.Atoi(strings.TrimSpace(vtemp[recoffset.duration_start:recoffset.duration_end]))
					fr1.calling_number = strings.TrimSpace(vtemp[recoffset.calling_number_start:recoffset.calling_number_end])
					fr1.called_number = strings.TrimSpace(vtemp[recoffset.called_number_start:recoffset.called_number_end])
					fr1.condition_code = strings.TrimSpace(vtemp[recoffset.condition_code_start:recoffset.condition_code_end])

					if ldebuggmode > 0 {
						fmt.Println("Condition code:", fr1.condition_code)
						fmt.Println("Test slice", vtemp[93:94])
					}

					_, err = stmt.Exec(
						fr1.dtime.Unix(),   // UNIX timestamp
						fr1.originaldt,     // оригинальная строка даты
						fr1.duration,       // длительность
						fr1.called_number,  // вызываемый номер
						fr1.calling_number, // вызывающий номер
						fr1.condition_code, // код состояния
					)

					if err != nil {
						fmt.Println("Ошибка выполнения INSERT:", err)
						continue
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
	}
}

func (srv *Server) ListenAndServe(ctx context.Context) error {
	addr := srv.Addr
	if addr == "" {
		addr = ":5001"
	}
	fmt.Println("Запущен сервис на:", addr)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Println("Ошибка:", err)
		return err
	}
	go func() {
		<-ctx.Done()
		listener.Close() // Разблокирует Accept()
	}()

	for {
		select {
		case <-ctx.Done():
			srv.wg.Wait()
			return ctx.Err()
		default:
			conn, err := listener.Accept()
			if err != nil {
				select {
				case <-ctx.Done():
					srv.wg.Wait()
					return ctx.Err()
				default:
					fmt.Println("Ошибка подключения:", err)
					continue
				}
			}
			fmt.Println("PowerCCDR подключение:", conn.RemoteAddr())
			srv.wg.Add(1)
			go srv.handle(ctx, conn)
		}
	}
}

// Описание JSON параметров для работы с базой данных
type params struct {
	Databasename     string `json:"dbname"`
	Databaseuser     string `json:"dbuser"`
	Databaseurl      string `json:"dburl"`
	Debuggmode       int    `json:"debugmode"`
	Sendsmsurl       string `json:"smsurl"`
	Companyname      string `json:"company"`
	DatetimeLocation string `json:"location"`
}

var recoffset CDR_Record_offsett
var db *sql.DB
var errsql error
var ldebuggmode int
var comname string
var SMSurl string
var Timezone string

const JsonFileName = "params.json"

func main() {
	recoffset = CDR_Record_offsett{0, 11, 12, 17, 18, 33, 34, 57, 93, 94}
	var JParams params
	// Открываем файл с настройками
	jSettingsFile, err := os.Open(JsonFileName)
	// Проверяем на ошибки
	if err != nil {
		fmt.Println("Ошибка:", err)
	}
	defer jSettingsFile.Close()

	FData, err := ioutil.ReadAll(jSettingsFile)
	if err != nil {
		fmt.Println("Ошибка:", err)
	}

	json.Unmarshal(FData, &JParams)
	ldebuggmode = JParams.Debuggmode
	comname = JParams.Companyname
	SMSurl = JParams.Sendsmsurl

	Timezone = "Europe/Moscow"
	if len(JParams.DatetimeLocation) != 0 {
		Timezone = JParams.DatetimeLocation
	}

	fmt.Println("Режим отладки:", strconv.Itoa(JParams.Debuggmode))

	//Получение пароля из KeyRing посредством запуска Python скрипта
	databasepasswordby, err := exec.Command("./getuser.py", "-u"+JParams.Databaseuser, "-k"+JParams.Databasename).Output()
	if err != nil {
		fmt.Println("Ошибка получения пароля для пользователя " + JParams.Databaseuser + " из KeyRing")
		os.Exit(1)
	}
	databasepassword := strings.TrimSpace(string(databasepasswordby))

	if databasepassword == "" {
		fmt.Println("Ошибка получения пароля для пользователя " + JParams.Databaseuser + " из KeyRing " + JParams.Databasename)
		os.Exit(1)
	}

	//Создаем строку для соединения с базой данных
	DsToLog := fmt.Sprintf("%s@tcp(%s)/%s", JParams.Databaseuser, JParams.Databaseurl, JParams.Databasename)
	DsStr := fmt.Sprintf("%s:%s@tcp(%s)/%s", JParams.Databaseuser, databasepassword, JParams.Databaseurl, JParams.Databasename)
	fmt.Println("Попытка соединения с базой данных:", DsToLog)

	db, errsql = sql.Open("mysql", DsStr)
	if errsql != nil {
		fmt.Println("Ошибка:", errsql)
		panic(errsql.Error())
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		fmt.Println("Ошибка:", err)
		panic(errsql.Error())
	}
	fmt.Println("Старт CDR сервиса")
	s1 := NewServer(":5001")

	//Ждем сигнал и завершаем все потоки
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGKILL)
	defer stop()
	s1.ListenAndServe(ctx)
}
