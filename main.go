package main

import (
	"GoRepo/deribitBot/pkg/urShadow/go-vk-api"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/adampointer/go-deribit"
	"github.com/adampointer/go-deribit/client/operations"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	_ "github.com/lib/pq"
)

// Переработать функции чтобыработали как в вк так и в тг
// /position проверка на существующие позиции. if Работает не корректно
func main() {
	api := vk.New("ru")
	err := api.Init("45072d656ce977e98e910ccb5fd07f2e4f312c4b0a5bc1aa6c8f68c9ec2bf49a919fd7968ed048ca5b89e")
	if err != nil {
		log.Fatalln(err)
	} else {
		fmt.Print("Auth vk OK \n")
	}
	bot, err := tgbotapi.NewBotAPI("855439174:AAFj2g1wvUJ0IoAxpCFfWmifj4pNcyVF_eQ")
	if err != nil {
		fmt.Printf("error: ", err)
	}
	log.Printf("Authorized on account %s", bot.Self.UserName)
	go tgReadMessage(bot)
	api.OnNewMessage(func(msg *vk.LPMessage) {
		go vkReadMessage(api, msg)
	})
	api.RunLongPoll()
}

//tg functions
func tgReadMessage(bot *tgbotapi.BotAPI) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates, err := bot.GetUpdatesChan(u)
	if err != nil {
		fmt.Println("tg update error: ", err)
	}
	for update := range updates {
		if update.Message == nil { // ignore any non-Message Updates
			continue
		}
		log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)
		key, skey := getKeys(0, update.Message.From.UserName) //get deribit keys
		if key != "" && skey != "" {
			e := authExchange(key, skey)
			switch {
			case update.Message.Text == "/help":
				str := `/add - 'key' 'skey'
				/position - current positions
				/balance - Current balance
				/alert BTC/ETH 6000 - make alert on currency
				/index - get current index
				/history - balance graph`
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, str)
				bot.Send(msg)
			case update.Message.Text == "/position":
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, getPosition(e))
				bot.Send(msg)
			case update.Message.Text == "/balance": //ПОСТРОИТЬ ГРАФИК
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, getBalance(e))
				bot.Send(msg)
			case strings.Contains(update.Message.Text, "/alert"):
				splitMsg := strings.Split(update.Message.Text, " ")
				if len(splitMsg) == 3 {
					price, err := strconv.ParseFloat(splitMsg[2], 64)
					if err != nil {
						msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Try again: /alert BTC/ETH 6000")
						bot.Send(msg)
					} else {
						msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Alert established at "+splitMsg[2])
						bot.Send(msg)
						go alertPrice(nil, "", update.Message.From.UserName, price, splitMsg[1], e)
					}
				} else {
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, "You wrote somethig wrong\n /alert BTC/ETH 6000")
					bot.Send(msg)
				}
			case update.Message.Text == "/index":
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Current mark price BTC: "+floatToString(getPrice(e, "BTC"))+"\nCurrent mark price ETH: "+floatToString(getPrice(e, "ETH")))
				bot.Send(msg)
			/*case msg.Text == "/history":
			go getBalanceHistory(e)*/
			case strings.Contains(update.Message.Text, "/add"):
				go tgAddUser(bot, update.Message.From.UserName, update.Message.Text, update.Message.Chat.ID)
			default:
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Unknown command")
				bot.Send(msg)
			}
		} else {
			if update.Message.Text == "/start" {
				str := `В первую очередь вам нужно добавить api ключи. Получить их можно здесь: https://www.deribit.com/main#/account?scrollTo=api
				Вопросы можно задать тут: https://t-do.ru/joinchat/Hk9v0RZZcRcvy-cDDL4_-A
				Бот умеет:
				/add - 'key' 'skey'
				/position - current positions 
				/balance - Current balance
				/alert BTC/ETH 6000 - make alert on currency
				/index - get current index
				/history - balance graph(Не работает на данный момент)
				`
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, str)
				bot.Send(msg)
			} else {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "First of all you need connect deribit with api\n")
				bot.Send(msg)
			}
		}

	}
}

func tgAddUser(bot *tgbotapi.BotAPI, name string, msg string, chatID int64) {
	str := strings.Split(msg, " ")
	switch {
	case len(str) < 3 || len(str) > 3:
		msg := tgbotapi.NewMessage(chatID, "You wrote something wrong.\n /add 'key' 'skey'")
		bot.Send(msg)
	case len(str) == 3:
		addUserDB("", name, str[1], str[2])
		msg := tgbotapi.NewMessage(chatID, "Account added")
		bot.Send(msg)
	}
}

//vk functions
func vkReadMessage(api *vk.VK, msg *vk.LPMessage) {
	if msg.Flags&vk.FlagMessageOutBox == 0 {
		id := strconv.FormatInt(msg.FromID, 10)
		if msg.Text == "/end" && msg.FromID == 88942272 {
			os.Exit(1)
		}
		if strings.Contains(msg.Text, "/add") {
			vkAddUser(api, msg.Text, id)
		} else {
			fmt.Println(id + " " + msg.Text)
			key, skey := getKeys(msg.FromID, "") //get deribit keys
			if key != "" && skey != "" {
				e := authExchange(key, skey)
				switch {
				case msg.Text == "/help":
					go vkSendMessage(api, `/add - 'key' 'skey'\n /position - current positions \n /balance - Current balance\n /alert BTC/ETH 6000 - make alert\n
				/index - get current index\n /history - balance graph`, id)
				case msg.Text == "/position":
					go vkSendMessage(api, getPosition(e), id)
				case msg.Text == "/balance": //ПОСТРОИТЬ ГРАФИК
					go vkSendMessage(api, getBalance(e), id)
				case strings.Contains(msg.Text, "/alert"):
					splitMsg := strings.Split(msg.Text, " ")
					if len(splitMsg) == 3 {
						price, err := strconv.ParseFloat(splitMsg[2], 64)
						if err != nil {
							vkSendMessage(api, "Try again: /alert BTC/ETH 6000", id)
						} else {
							go vkSendMessage(api, "Alert established at "+splitMsg[2], id)
							go alertPrice(api, id, "", price, splitMsg[1], e)
						}
					}
				case msg.Text == "/index":
					go vkSendMessage(api, "Current mark price BTC: "+floatToString(getPrice(e, "BTC"))+"\nCurrent mark price ETH: "+floatToString(getPrice(e, "ETH")), id)
				/*case msg.Text == "/history":
				go getBalanceHistory(e)*/
				case strings.Contains(msg.Text, "/add"):
					go vkAddUser(api, msg.Text, id)
				default:
					vkSendMessage(api, "Unknown command", id)
				}
			} else {
				if msg.Text == "/start" {
					str := `В первую очередь вам нужно добавить api ключи. Получить их можно здесь: https://www.deribit.com/main#/account?scrollTo=api
					Вопросы можно задать тут: https://t-do.ru/joinchat/Hk9v0RZZcRcvy-cDDL4_-A
					Бот умеет:
					/add - 'key' 'skey'
					/position - current positions 
					/balance - Current balance
					/alert BTC/ETH 6000 - make alert on currency
					/index - get current index
					/history - balance graph(Не работает на данный момент)
					`
					go vkSendMessage(api, str, id)
				} else {
					go vkSendMessage(api, "First of all you need connect deribit with api\n /add - 'key' 'skey'", id)
				}
			}
		}
	}
}

func vkAddUser(api *vk.VK, msg string, id string) {
	str := strings.Split(msg, " ")
	switch {
	case len(str) < 3 || len(str) > 3:
		go vkSendMessage(api, "You wrote something wrong.\n /add 'key' 'skey'", id)
	case len(str) == 3:
		addUserDB(id, "", str[1], str[2])
		go vkSendMessage(api, "Account added", id)
	}
}

func vkSendMessage(api *vk.VK, msg string, id string) {
	api.Messages.Send(vk.RequestParams{
		"peer_id": id,
		"message": msg,
	})
}

//Database functions
func getKeys(id int64, name string) (string, string) {
	Addr := "user=postgres dbname=ExchangeBot host=127.0.0.1 port=5432 sslmode=disable"
	db, err := sql.Open("postgres", Addr)
	if err != nil {
		fmt.Println("Cant open DB: ", err)
	}
	err = db.Ping()
	if err != nil {
		fmt.Println("Connect to DB err: ", err)
	}
	switch {
	case id != 0:
		row := db.QueryRow("SELECT deribitkey FROM users WHERE vkid = $1", id)
		var key string
		err = row.Scan(&key)
		if err == sql.ErrNoRows {
			fmt.Println("Не найдено записей")
		}
		row = db.QueryRow("SELECT deribitskey FROM users WHERE vkid = $1", id)
		var skey string
		err = row.Scan(&skey)
		if err == sql.ErrNoRows {
			fmt.Println("Не найдено записей")
		}
		db.Close()
		return key, skey
	case name != "":
		row := db.QueryRow("SELECT deribitkey FROM users WHERE tgname = $1", name)
		var key string
		err = row.Scan(&key)
		if err == sql.ErrNoRows {
			fmt.Println("Не найдено записей")
		}
		row = db.QueryRow("SELECT deribitskey FROM users WHERE tgname = $1", name)
		var skey string
		err = row.Scan(&skey)
		if err == sql.ErrNoRows {
			fmt.Println("Не найдено записей")
		}
		db.Close()
		return key, skey
	}
	return "", ""
}

func addUserDB(id string, name string, key string, skey string) {
	Addr := "user=postgres dbname=ExchangeBot host=127.0.0.1 port=5432 sslmode=disable"
	db, err := sql.Open("postgres", Addr)
	if err != nil {
		fmt.Println("Cant open DB: ", err)
	}
	err = db.Ping()
	if err != nil {
		fmt.Println("Connect to DB err: ", err)
	}
	//check if user already added in another messenger

	switch {
	case name != "": //tg add user without vkid
		var lastID int
		query := "INSERT INTO users(vkid, tgname, deribitkey, deribitskey) VALUES('', $1, $2, $3) RETURNING id"
		row := db.QueryRow(query, name, key, skey)
		err = row.Scan(&lastID)
		fmt.Println(lastID)
		fmt.Println(err)
		db.Close()
	case id != "": //vk add user without tgname
		var lastID int
		query := "INSERT INTO users(vkid, tgname, deribitkey, deribitskey) VALUES($1, '', $2, $3) RETURNING id"
		i, _ := strconv.Atoi(id)
		row := db.QueryRow(query, i, key, skey)
		err = row.Scan(&lastID)
		fmt.Println(lastID)
		fmt.Println(err)
		db.Close()
	}
}

//Deribit
func authExchange(key string, skey string) *deribit.Exchange {
	errs := make(chan error)
	stop := make(chan bool)
	e, err := deribit.NewExchange(false, errs, stop)
	if err != nil {
		fmt.Println("Error creating connection: ", err)
	}
	if err := e.Connect(); err != nil {
		fmt.Println("Error connecting to exchange: ", err)
	}
	go func() {
		err := <-errs
		stop <- true
		fmt.Println("RPC error: ", err)
	}()
	client := e.Client()
	res, err := client.GetPublicTest(&operations.GetPublicTestParams{})
	if err != nil {
		fmt.Println("Error testing connection: ", err)
	}
	log.Printf("Connected to Deribit API v%s", *res.Payload.Result.Version)
	if err := e.Authenticate(key, skey); err != nil {
		fmt.Println("Error authenticating: ", err)
	}
	return e
}

func alertPrice(api *vk.VK, id string, name string, price float64, instrument string, e *deribit.Exchange) {
	index := getPrice(e, instrument)
	if price < index {
		for {
			index = getPrice(e, instrument)
			if price >= index {
				if api != nil {
					go vkSendMessage(api, "Alert on price "+floatToString(price), id)
					break
				} else {

					break
				}
			}
			time.Sleep(time.Second * 20)
		}
	} else {
		for {
			index = getPrice(e, instrument)
			if price <= index {
				if api != nil {
					go vkSendMessage(api, "Alert on price "+floatToString(price), id)
					break
				} else {

					break
				}
			}
			time.Sleep(time.Second * 20)
		}
	}
}

func getPrice(e *deribit.Exchange, instrument string) float64 {
	client := e.Client()
	indexBTC, err := client.GetPublicGetIndex(&operations.GetPublicGetIndexParams{Currency: "BTC"})
	if err != nil {
		fmt.Println("Error getting index: ", err)
	}
	indexETH, err := client.GetPublicGetIndex(&operations.GetPublicGetIndexParams{Currency: "ETH"})
	if err != nil {
		fmt.Println("Error getting index: ", err)
	}
	switch {
	case instrument == "BTC":
		return *indexBTC.Payload.Result.BTC
	case instrument == "ETH":
		return indexETH.Payload.Result.ETH
	}
	return 1.0
}

func getPosition(e *deribit.Exchange) string {
	client := e.Client()
	//get positions for BTC
	var str string
	positionsBTC, err := client.GetPrivateGetPositions(&operations.GetPrivateGetPositionsParams{Currency: "BTC"})
	if err != nil {
		fmt.Println("Error getting positions: ", err)
	}
	//Переработать. Хуйня какая то!!
	if len(positionsBTC.Payload.Result) > 0 {
		if *positionsBTC.Payload.Result[0].AveragePrice > 0.0 {
			str = str + "BTC"
			for i := 0; i < len(positionsBTC.Payload.Result); i++ {
				avg := *positionsBTC.Payload.Result[i].AveragePrice
				index := positionsBTC.Payload.Result[i].IndexPrice
				eqt := positionsBTC.Payload.Result[i].Size
				pnl := *positionsBTC.Payload.Result[i].FloatingProfitLoss
				usd := pnl * getPrice(e, "BTC")
				str = str + "\nCount: " + floatToString(*eqt) + "\nEntry: " + floatToString(avg) + "\nIndex: " + floatToString(*index) + "\nProfitLoss: BTC " + floatToString(pnl) + "  $" + floatToString(usd) + " ₽" + floatToString(usd*65)
			}
		} else {
			str = str + "\nNo positions BTC"
		}
	} else {
		str = str + "\nNo positions BTC"
	}
	//get positions for ETH
	positionsETH, err := client.GetPrivateGetPositions(&operations.GetPrivateGetPositionsParams{Currency: "ETH"})
	if err != nil {
		fmt.Println("Error getting positions: ", err)
	}
	if len(positionsETH.Payload.Result) > 0 {
		str = str + "ETH"
		for i := 0; i < len(positionsETH.Payload.Result); i++ {
			fmt.Println(positionsETH)
			avg := *positionsETH.Payload.Result[i].AveragePrice
			index := positionsETH.Payload.Result[i].IndexPrice
			eqt := positionsETH.Payload.Result[i].Size
			pnl := *positionsETH.Payload.Result[i].FloatingProfitLoss
			usd := pnl * getPrice(e, "ETH")
			str = str + "\nCount: " + floatToString(*eqt) + "\nEntry: " + floatToString(avg) + "\nIndex: " + floatToString(*index) + "\nProfitLoss: BTC " + floatToString(pnl) + "  $" + floatToString(usd) + " ₽" + floatToString(usd*65)
		}
	} else {
		str = str + "\nNo positions ETH"
	}
	return str
}

func getBalance(e *deribit.Exchange) string {
	client := e.Client()
	// Account summary BTC
	summaryBTC, err := client.GetPrivateGetAccountSummary(&operations.GetPrivateGetAccountSummaryParams{Currency: "BTC"})
	if err != nil {
		fmt.Println("Error getting account summary BTC: ", err)
	}
	balanceBTC := *summaryBTC.Payload.Result.Equity
	usdB := balanceBTC * getPrice(e, "BTC")
	// Account summary ETH
	summaryETH, err := client.GetPrivateGetAccountSummary(&operations.GetPrivateGetAccountSummaryParams{Currency: "ETH"})
	if err != nil {
		fmt.Println("Error getting account summary ETH: ", err)
	}
	balanceETH := *summaryETH.Payload.Result.Equity
	usdE := balanceETH * getPrice(e, "ETH")
	return "BTC " + floatToString(balanceBTC) + " $" + floatToString(usdB) + " ₽" + floatToString(usdB*65) + "\nETH " + floatToString(balanceETH) + " $" + floatToString(usdE) + " ₽" + floatToString(usdE*65)
}

/* Not working
func getBalanceHistory(e *deribit.Exchange) {
	client := e.Client()
	// Account settlement history
	history, err := client.GetPrivateGetSettlementHistoryByCurrency(&operations.GetPrivateGetSettlementHistoryByCurrencyParams{Currency: "BTC"})
	if err != nil {
		fmt.Println("Error getting account settlement: ", err)
	}
	fmt.Println(history.Payload.Result)
}
*/
//Some Help
func strPointer(str string) *string {
	return &str
}

func floatToString(inputnum float64) string {
	// to convert a float number to a string
	return strconv.FormatFloat(inputnum, 'f', 6, 64)
}
