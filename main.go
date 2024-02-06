package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/joho/godotenv"
)

type TimeSchedule struct {
	Id          string
	RangeToText string
	CanSelect   bool
}

type ScheduleCheck struct {
	Id      string
	Checked bool
}

func isHalfWidthDigitString(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func inputWithPrompt(promptText string) string {
	var input string

	for {
		fmt.Print(promptText)
		fmt.Scan(&input)
		if input != "" && isHalfWidthDigitString(input) {
			break
		} else {
			fmt.Println("半角数字で入力してください。")
		}
	}
	return input
}

func ensureTwoDigits(s string) string {
	if len(s) == 1 {
		return "0" + s
	}
	return s
}

func isValidDate(year, month, day int) bool {
	t := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local)
	return t.Year() == year && t.Month() == time.Month(month) && t.Day() == day
}

func askAboutReserveDate() (yearMonthStr, dateStr string) {
	fmt.Println("予約する日付を入力してください。（半角数字のみ）")

	yearStr := inputWithPrompt("年: ")
	monthStr := inputWithPrompt("月: ")
	dateStr = inputWithPrompt("日: ")

	yearInt, _ := strconv.Atoi(yearStr)
	monthInt, _ := strconv.Atoi(monthStr)
	dateInt, _ := strconv.Atoi(dateStr)

	if !isValidDate(yearInt, monthInt, dateInt) {
		fmt.Println("無効な日付です。正しい日付を入力してください。")
		return
	}

	// 入力した日付が過去の日付でないかチェック
	inputDate := time.Date(yearInt, time.Month(monthInt), dateInt, 0, 0, 0, 0, time.Local)

	// 時・分・秒を0にして日付を比較
	now := time.Now()
	currentDateRounded := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	isTodayOrLater := inputDate.Compare(currentDateRounded)

	if isTodayOrLater == -1 {
		fmt.Println("過去の日付を入力することはできません。")
	}

	// 3ヶ月後までが有効な予約日
	afterThreeMonth := currentDateRounded.AddDate(0, 3, -1)

	isWithInThreeMonth := inputDate.Compare(afterThreeMonth)

	if isWithInThreeMonth == 1 {
		fmt.Println("3ヶ月より先の日付を入力することはできません。")
	}

	monthStr = ensureTwoDigits(monthStr)
	dateStr = ensureTwoDigits(dateStr)

	yearMonthStr = yearStr + monthStr

	return yearMonthStr, dateStr
}

func init() {
	// JSTタイムゾーンに設定
	loc, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		panic(err)
	}
	time.Local = loc
}

func main() {
	// .envファイルから環境変数を読み込む
	if err := godotenv.Load(); err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	// 環境変数を使用
	LoginURL := os.Getenv("LOGIN_URL")
	LoginID := os.Getenv("LOGIN_ID")
	Password := os.Getenv("PASSWORD")
	CardNumber := os.Getenv("CARD_NUMBER")
	SecurityCode := os.Getenv("SECURITY_CODE")

	// yyyymm, ddの形式で取得する
	reserveYearMonth, reserveDate := askAboutReserveDate()

	// headlessフラグをfalseにしてブラウザを表示する
	// ChromeのWelcomeページを非表示にする
	allocCtx, cancel := chromedp.NewExecAllocator(
		context.Background(),
		chromedp.Flag("headless", false),
		chromedp.NoFirstRun,
	)
	defer cancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	if err := chromedp.Run(ctx,
		// ホームページに遷移する
		chromedp.Navigate(LoginURL),

		// ホームページからログインページに遷移する
		chromedp.WaitVisible(`body`, chromedp.ByQuery),
		chromedp.Click(`input[value="ログイン"]`, chromedp.ByQuery),

		// ログイン処理
		chromedp.WaitVisible(`body`, chromedp.ByQuery),
		chromedp.SendKeys(`input[name="in_member"]`, LoginID, chromedp.ByQuery),
		chromedp.SendKeys(`input[name="in_mpassword"]`, Password, chromedp.ByQuery),
		chromedp.Click(`input[name="ログイン"][type="submit"]`, chromedp.ByQuery),

		// selectタグから月を選択
		chromedp.WaitVisible(`body`, chromedp.ByQuery),
		chromedp.SetAttributeValue(fmt.Sprintf(`select.pull_160[name="Ym_select"] > option[value="%s"]`, reserveYearMonth), "selected", "selected", chromedp.ByQuery),
		chromedp.Click(`input.btn_kousin[name="button"][type="submit"]`, chromedp.ByQuery),

		// カレンダーから日付を選択
		chromedp.WaitVisible(`body`, chromedp.ByQuery),
		chromedp.Click(fmt.Sprintf(`input[type="button"][name="%s"]`, reserveDate), chromedp.ByQuery),
	); err != nil {
		log.Fatalf("Failed to complete login process: %v", err)
	}

	// 予約したい枠を定義
	schedules := []TimeSchedule{
		{
			Id:          "0230",
			RangeToText: "20:30 ~ 21:00",
			CanSelect:   false,
		},
		{
			Id:          "0231",
			RangeToText: "21:00 ~ 21:30",
			CanSelect:   false,
		},
		{
			Id:          "0232",
			RangeToText: "21:30 ~ 22:00",
			CanSelect:   false,
		},
		{
			Id:          "0233",
			RangeToText: "22:00 ~ 22:30",
			CanSelect:   false,
		},
	}

	reserveFail := false

	// 予約可能かチェック
	for _, schedule := range schedules {
		if err := chromedp.Run(ctx,
			chromedp.Sleep(1*time.Second),
			chromedp.Evaluate(fmt.Sprintf(`!!document.querySelector('tr:nth-child(3) input[value="%s"]')`, schedule.Id), &schedule.CanSelect),
		); err != nil {
			log.Println("Failed to check if the checkbox is exist")
		}
		if !schedule.CanSelect {
			errorMessage := fmt.Sprintf("【エラー：他のユーザーにより予約済み】%s枠", schedule.RangeToText)
			fmt.Println(errorMessage)
			reserveFail = true
		}
	}

	// 予約済みの枠がある場合はエラーを出力して終了
	if reserveFail {
		log.Fatal("【エラー：他のユーザーにより予約済み】枠が埋まっているため予約できません。")
	}

	// 予約IDだけを配列に格納
	reserveIds := make([]string, len(schedules))
	for i, schedule := range schedules {
		reserveIds[i] = schedule.Id
	}

	// 時間帯を選択してチェックボックスにチェックを入れる
	for _, reserveId := range reserveIds {
		if err := chromedp.Run(ctx,
			chromedp.Click(fmt.Sprintf(`tr:nth-child(3) input[value="%s"]`, reserveId), chromedp.ByQuery),
		); err != nil {
			log.Fatal("Failed to check", err)
		}
	}

	// チェックしたか確認
	scheduleChecks := []ScheduleCheck{
		{
			Id:      "0230",
			Checked: false,
		},
		{
			Id:      "0231",
			Checked: false,
		},
		{
			Id:      "0232",
			Checked: false,
		},
		{
			Id:      "0233",
			Checked: false,
		},
	}

	checkFail := false

	// チェックできているか確認
	for _, scheduleCheck := range scheduleChecks {
		if err := chromedp.Run(ctx,
			chromedp.Evaluate(fmt.Sprintf(`document.querySelector('tr:nth-child(3) input[value="%s"]').checked`, scheduleCheck.Id), &scheduleCheck.Checked),
		); err != nil {
			log.Println("Failed to check if the checkbox is exist")
		}
		if !scheduleCheck.Checked {
			checkFail = true
		}
	}

	// チェックに失敗した場合はエラーを出力して終了
	if checkFail {
		log.Fatal("【エラー：チェックに失敗】")
	}

	// 予約ボタンをクリック
	if err := chromedp.Run(ctx,
		chromedp.Click(`input.btn_yoyaku[name="yoyaku_btn"]`, chromedp.ByQuery),
	); err != nil {
		log.Fatal("Failed to check", err)
	}

	// 予約内容のフォーム入力
	if err := chromedp.Run(ctx,
		// 人数: 3人
		chromedp.SendKeys(`input[name="ninzu"]`, "3", chromedp.ByQuery),

		// 予約区分: バンド練習
		chromedp.Click(`input[name="tokutei_flg"][value="0"]`, chromedp.ByQuery),

		// 支払い方法: オンライン決済
		chromedp.Click(`input[name="s_pay"][value="3"]`, chromedp.ByQuery),

		// 楽器タブを選択
		chromedp.Click(`a[href="#page5"]`, chromedp.ByQuery),

		// ストラトとジャズベにチェック
		chromedp.Click(`input#bihin11`, chromedp.ByQuery),
		chromedp.Click(`input#bihin14`, chromedp.ByQuery),

		// 次へボタンをクリック
		chromedp.Click(`input[type="button"][value="次へ"]`, chromedp.ByQuery),

		// 最終確認ボタンをクリック
		chromedp.Click(`input[type="button"][value="最終確認"]`, chromedp.ByQuery),

		// 予約するボタンをクリック
		chromedp.Click(`input[type="button"][value="予約する"]`, chromedp.ByQuery),

		// 決済ボタンが表示されるまで待機
		chromedp.WaitVisible(`input[type="submit"][value="決済手続きへ"]`, chromedp.ByQuery),

		// 決済手続きへボタンをクリック
		chromedp.WaitVisible(`input[type="submit"][value="決済手続きへ"]`, chromedp.ByQuery),
		chromedp.Click(`input[type="submit"][value="決済手続きへ"]`, chromedp.ByQuery),

		// 「クレジットカード」をクリック
		chromedp.Click(`input#paytype_credit`, chromedp.ByQuery),

		// 「進む」をクリック
		chromedp.Click(`input[type="submit"][value="進む"]`, chromedp.ByQuery),

		// 「お支払い方法」を「一括」に設定
		chromedp.SetAttributeValue(`select#__pay_method_list > option[value="1"]`, "selected", "selected", chromedp.ByQuery),

		// カード番号を入力
		chromedp.SendKeys(`input#Name`, CardNumber, chromedp.ByQuery),

		// 有効期限を入力：月
		chromedp.SetAttributeValue(`select#__expire_month_list > option[value="10"]`, "selected", "selected", chromedp.ByQuery),

		// 有効期限を入力：年
		chromedp.SetAttributeValue(`select#__expire_year_list > option[value="26"]`, "selected", "selected", chromedp.ByQuery),

		// セキュリティコードを入力
		chromedp.SendKeys(`input#SecurityCode`, SecurityCode, chromedp.ByQuery),

		// 「確認する」ボタンをクリック
		chromedp.Click(`input[type="submit"][value="確認する"]`, chromedp.ByQuery),

		// 「決済する」ボタンをクリック
		chromedp.Click(`input[type="submit"][value="決済する"]`, chromedp.ByQuery),
	); err != nil {
		log.Fatal("Failed to reserve", err)
	}

	log.Println("Login process completed. Keeping the window open...")
	select {}
}
