package main

import (
	"bufio"
	"bytes"
	"encoding/xml"
	"fmt"
	"github.com/go-playground/validator/v10"
	"golang.org/x/net/html/charset"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type ValCurs struct {
	Date string `xml:"Date,attr"`
	Rate []Rate `xml:"Valute"`
}

type Rate struct {
	ID        string `xml:"ID,attr"`
	NumCode   string `xml:"NumCode"`
	CharCode  string `xml:"CharCode"`
	Nominal   string `xml:"Nominal"`
	Name      string `xml:"Name"`
	Value     string `xml:"Value"`
	VunitRate string `xml:"VunitRate"`
}

type CurrencyRate struct {
	Date     time.Time
	Code     string
	Name     string
	Nominal  int
	Value    float64
	UnitRate float64
}

func main() {
	validate := validator.New(validator.WithRequiredStructEnabled())
	var date string
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("Введите дату в формате DD/MM/YYYY (или нажмите Enter для текущей даты): ")
	scanner.Scan()
	date = scanner.Text()
	if len(date) == 0 {
		date = time.Now().Format("02/01/2006")
	} else {
		err := validate.Var(date, "datetime=02/01/2006")
		if err != nil {
			log.Fatal("Неверный формат даты. Требуется - DD/MM/YYYY")
		}
	}
	rates, err := getCurrenciesForLastDays(date, 90)
	if err != nil {
		log.Fatalf("Ошибка при получении данных: %v", err)
	}

	if len(rates) == 0 {
		log.Fatal("Не найдено данных о курсах валют")
	}

	maxRate, minRate, avgRate := analyzeRates(rates)

	fmt.Println("\nРезультаты:")
	fmt.Printf("Максимальный курс: %f RUB за 1 %s (%s) приходится на дату %s с номиналом %d \n",
		maxRate.UnitRate, maxRate.Name, maxRate.Code, maxRate.Date.Format("02/01/2006"), maxRate.Nominal)
	fmt.Printf("Минимальный курс: %f RUB за 1 %s (%s) на дату %s с номиналом %d \n",
		minRate.UnitRate, minRate.Name, minRate.Code, minRate.Date.Format("02/01/2006"), minRate.Nominal)
	fmt.Printf("Средний курс по всем валютам: %f RUB\n", avgRate)
}

func getCurrenciesForLastDays(startDate string, days int) ([]CurrencyRate, error) {
	var allRates []CurrencyRate
	baseURL := "https://www.cbr.ru/scripts/XML_daily_eng.asp"

	start, err := time.Parse("02/01/2006", startDate)
	if err != nil {
		return nil, fmt.Errorf("ошибка парсинга даты: %v", err)
	}

	client := &http.Client{}

	for i := 0; i < days; i++ {
		date := start.AddDate(0, 0, -i)
		dateStr := date.Format("02/01/2006")

		params := url.Values{}
		params.Add("date_req", dateStr)

		req, err := http.NewRequest("GET", baseURL+"?"+params.Encode(), nil)
		if err != nil {
			return nil, fmt.Errorf("ошибка создания запроса для даты %s: %v", dateStr, err)
		}

		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")

		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("ошибка запроса для даты %s: %v", dateStr, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("неверный статус код для даты %s: %d", dateStr, resp.StatusCode)
		}

		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)

		decoder := xml.NewDecoder(buf)
		decoder.CharsetReader = charset.NewReaderLabel

		var valCurs ValCurs
		if err := decoder.Decode(&valCurs); err != nil {
			return nil, fmt.Errorf("ошибка декодирования XML для даты %s: %v", dateStr, err)
		}

		for _, valute := range valCurs.Rate {
			RateDate, _ := time.Parse("02.01.2006", valCurs.Date)
			nominal, _ := strconv.Atoi(valute.Nominal)
			value := strings.ReplaceAll(valute.Value, ",", ".")
			FloatValue, _ := strconv.ParseFloat(value, 64)

			unitRate := strings.ReplaceAll(valute.VunitRate, ",", ".")
			FloatUnitRate, _ := strconv.ParseFloat(unitRate, 64)
			rate := CurrencyRate{
				Date:     RateDate,
				Name:     valute.Name,
				Code:     valute.CharCode,
				Nominal:  nominal,
				Value:    FloatValue,
				UnitRate: FloatUnitRate,
			}
			allRates = append(allRates, rate)
		}

	}

	return allRates, nil
}

func analyzeRates(rates []CurrencyRate) (maxRate, minRate CurrencyRate, avgRate float64) {
	if len(rates) == 0 {
		return
	}

	maxRate = rates[0]
	minRate = rates[0]
	sum := 0.0
	count := 0

	for _, rate := range rates {
		if rate.UnitRate > maxRate.UnitRate {
			maxRate = rate
		}
		if rate.UnitRate < minRate.UnitRate {
			minRate = rate
		}
		sum += rate.UnitRate
		count++
	}

	if count > 0 {
		avgRate = sum / float64(count)
	}
	return
}
