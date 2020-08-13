package main

import (
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
	"github.com/urShadow/go-vk-api"
)

// /position проверка на существующие позиции. if Работает не корректно
func main() {
	api := vk.New("ru")
	err := api.Init("")
	if err != nil {
		log.Fatalln(err)
	} else {
		fmt.Print("Auth vk OK \n")
	}
	// Bot init
	//Without SOCKS5
	bot, err := tgbotapi.NewBotAPI("")
	//tg proxy SOCKS5
	/*tr := http.Transport{
		DialContext: func(_ context.Context, network, addr string) (net.Conn, error) {
			socksDialer, err := proxy.SOCKS5("tcp", "2.234.226.32:17779", &proxy.Auth{}, proxy.Direct)
			if err != nil {
				return nil, err
			}
			return socksDialer.Dial(network, addr)
		},
	}
	bot, err := tgbotapi.NewBotAPIWithClient("", &http.Client{
		Transport: &tr,
	})*/
	if err != nil {

		log.Fatal(err)

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
		if strings.Contains(update.Message.Text, "/add") {
			go tgAddUser(bot, update.Message.From.UserName, update.Message.Text, update.Message.Chat.ID)
		} else {
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
							go alertPrice(nil, "", bot, update.Message.Chat.ID, price, splitMsg[1], e)
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
				default:
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Unknown command")
					bot.Send(msg)
				}
			} else {
				if update.Message.Text == "/start" {
					str := `В первую очередь вам нужно добавить api ключи. Получить их можно здесь: https://www.deribit.com/main#/account?scrollTo=api, затем использовать команду: /add 'key' 'skey'
					Также доступна группа вк: https://vk.com/club176452271 Ключи добавлять нужно на каждой платформе отдельно!
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
							go alertPrice(api, id, nil, 0, price, splitMsg[1], e)
						}
					} else {
						vkSendMessage(api, "Try again: /alert BTC/ETH 6000", id)
					}
				case msg.Text == "/index":
					go vkSendMessage(api, "Current mark price BTC: "+floatToString(getPrice(e, "BTC"))+"\nCurrent mark price ETH: "+floatToString(getPrice(e, "ETH")), id)
				/*case msg.Text == "/history":
				go getBalanceHistory(e)*/
				default:
					vkSendMessage(api, "Unknown command", id)
				}
			} else {
				switch {
				case msg.Text == "/start":
					str := `В первую очередь вам нужно добавить api ключи. Получить их можно здесь: https://www.deribit.com/main#/account?scrollTo=api, затем использовать команду: /add 'key' 'skey'
					Также доступна группа вк: https://vk.com/club176452271 Ключи добавлять нужно на каждой платформе отдельно!
					Бот умеет:
					/add - 'key' 'skey'
					/position - current positions 
					/balance - Current balance
					/alert BTC/ETH 6000 - make alert on currency
					/index - get current index
					/history - balance graph(Не работает на данный момент)
					`
					go vkSendMessage(api, str, id)
				default:
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
			fmt.Println("There is no keys in database")
		}
		row = db.QueryRow("SELECT deribitskey FROM users WHERE vkid = $1", id)
		var skey string
		err = row.Scan(&skey)
		if err == sql.ErrNoRows {
			fmt.Println("There is no keys in database")
		}
		db.Close()
		return key, skey
	case name != "":
		row := db.QueryRow("SELECT deribitkey FROM users WHERE tgname = $1", name)
		var key string
		err = row.Scan(&key)
		if err == sql.ErrNoRows {
			fmt.Println("There is no secret keys in database")
		}
		row = db.QueryRow("SELECT deribitskey FROM users WHERE tgname = $1", name)
		var skey string
		err = row.Scan(&skey)
		if err == sql.ErrNoRows {
			fmt.Println("There is no secret keys in database")
		}
		fmt.Println(key + " " + skey)
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
	switch {
	case name != "":
		var checkID int
		row := db.QueryRow("SELECT id FROM users WHERE deribitkey = $1", key)
		err := row.Scan(&checkID)
		if err == sql.ErrNoRows {
			query := "INSERT INTO users(vkid, tgname, deribitkey, deribitskey) VALUES($1, $2, $3, $4) RETURNING id" //tg add user without vkid
			db.QueryRow(query, 0, name, key, skey)
		} else {
			query := "UPDATE users SET tgname = $1 WHERE id = $2" //if row exist add tgname to row
			fmt.Println("Update")
			db.Exec(query, name, checkID)
		}
		db.Close()
	case id != "": //vk add user without tgname
		var checkID int
		row := db.QueryRow("SELECT id FROM users WHERE deribitkey = $1", key)
		fmt.Println("id")
		err := row.Scan(&checkID)
		if err == sql.ErrNoRows {
			//New user with new keys
			query := "INSERT INTO users(vkid, tgname, deribitkey, deribitskey) VALUES($1, '', $2, $3) RETURNING id" //vk add user without tgname
			i, _ := strconv.Atoi(id)
			fmt.Println("Insert")
			db.QueryRow(query, i, key, skey)
		} else {
			query := "UPDATE users SET vkid = $1 WHERE id = $2" //add vkid if row exist
			fmt.Println("Update")
			db.Exec(query, id, checkID)
		}
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

func alertPrice(api *vk.VK, id string, bot *tgbotapi.BotAPI, chatID int64, price float64, instrument string, e *deribit.Exchange) string {
	index := getPrice(e, instrument)
	if price < index {
		fmt.Println("Price < index")
		for {
			index = getPrice(e, instrument)
			fmt.Println("Cicle " + floatToString(index))
			if price >= index {
				if api != nil {
					go vkSendMessage(api, "Alert on price "+floatToString(price), id)
					break
				} else {
					msg := tgbotapi.NewMessage(chatID, "Alert on price "+floatToString(price))
					bot.Send(msg)
					break
				}
			}
			if price-index >= 1000 {
				return "Price is higher on 1000 usd. Alert disposed"
			}
			time.Sleep(time.Second * 20)
		}
	} else {
		fmt.Println("Price > index")
		for {
			index = getPrice(e, instrument)
			fmt.Println("Cicle " + floatToString(index))
			if price <= index {
				if api != nil {
					go vkSendMessage(api, "Alert on price "+floatToString(price), id)
					break
				} else {
					msg := tgbotapi.NewMessage(chatID, "Alert on price "+floatToString(price))
					bot.Send(msg)
					break
				}
			}
			if index-price >= 1000 {
				return "Price is lower on 1000 usd. Alert disposed"
			}
			time.Sleep(time.Second * 20)
		}
	}
	return ""
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
	case instrument == "BTC" || instrument == "btc":
		return *indexBTC.Payload.Result.BTC
	case instrument == "ETH" || instrument == "eth":
		return indexETH.Payload.Result.ETH
	}
	return 1.0
}

func getPosition(e *deribit.Exchange) string {
	client := e.Client()
	//get positions for BTC
	var str string
	var counter = false
	positionsBTC, err := client.GetPrivateGetPositions(&operations.GetPrivateGetPositionsParams{Currency: "BTC"})
	if err != nil {
		fmt.Println("Error getting positions BTC: ", err)
		str = str + "\nNo positions BTC"
	}
	if len(positionsBTC.Payload.Result) > 0 {
		str = str + "BTC\n"
		for i := 0; i < len(positionsBTC.Payload.Result); i++ {
			if *positionsBTC.Payload.Result[i].Size != 0.0 {
				ins := positionsBTC.Payload.Result[i].InstrumentName
				avg := *positionsBTC.Payload.Result[i].AveragePrice
				index := positionsBTC.Payload.Result[i].MarkPrice
				liq := positionsBTC.Payload.Result[i].EstimatedLiquidationPrice
				eqt := positionsBTC.Payload.Result[i].Size
				pnl := *positionsBTC.Payload.Result[i].TotalProfitLoss
				usd := pnl * getPrice(e, "BTC")
				counter = true
				str = str + "\nInstrument: " + ins + "\nCount: " + floatToString(*eqt) + "\nEntry: " + floatToString(avg) + "\nIndex: " + floatToString(*index) + "\nEst. Liq. Price: " + floatToString(liq) + "\nPL: BTC " + floatToString(pnl) + "  $" + floatToString(usd) + " ₽" + floatToString(usd*65) + "\n"
			}
		}
		if counter == false {
			str = str + "\nNo positions BTC"
		}
	} else {
		str = str + "\nNo positions BTC"
	}
	//get positions for ETH
	counter = false
	positionsETH, err := client.GetPrivateGetPositions(&operations.GetPrivateGetPositionsParams{Currency: "ETH"})
	if err != nil {
		fmt.Println("Error getting positions ETH: ", err)
		str = str + "\nNo positions ETH"
	} else {
		if len(positionsETH.Payload.Result) > 0 {
			str = str + "\nETH\n"
			for i := 0; i < len(positionsETH.Payload.Result); i++ {
				if *positionsETH.Payload.Result[i].Size != 0.0 {
					ins := positionsETH.Payload.Result[i].InstrumentName
					avg := *positionsETH.Payload.Result[i].AveragePrice
					index := *positionsETH.Payload.Result[i].MarkPrice
					liq := positionsETH.Payload.Result[i].EstimatedLiquidationPrice
					eqt := positionsETH.Payload.Result[i].Size
					pnl := *positionsETH.Payload.Result[i].FloatingProfitLoss
					usd := pnl * getPrice(e, "ETH")
					counter = true
					str = str + "\nInstrument: " + ins + "\nCount: " + floatToString(*eqt) + "\nEntry: " + floatToString(avg) + "\nIndex: " + floatToString(index) + "\nEst. Liq. Price: " + floatToString(liq) + "\nPL: ETH " + floatToString(pnl) + "  $" + floatToString(usd) + " ₽" + floatToString(usd*65) + "\n"
				}
			}
			if counter == false {
				str = str + "\nNo positions ETH"
			}
		}
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
