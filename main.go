package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/joho/godotenv"
)

func inputWithPrompt(promptText string) string {
	var input string
	fmt.Print(promptText)
	fmt.Scan(&input)
	return input
}

func ensureTwoDigits(s string) string {
	if len(s) == 1 {
		return "0" + s
	}
	return s
}

func askAboutReserveDate() (yearMonth, date string) {
	fmt.Println("予約する日付を入力してください。（半角数字）")

	year := inputWithPrompt("年: ")
	month := inputWithPrompt("月: ")
	date = inputWithPrompt("日: ")

	month = ensureTwoDigits(month)
	date = ensureTwoDigits(date)

	yearMonth = year + month

	return yearMonth, date
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

	// 枠が空いているかを確認
	var isExist1 bool
	var isExist2 bool
	var isExist3 bool
	var isExist4 bool
	if err := chromedp.Run(ctx,
		chromedp.Sleep(1*time.Second),
		chromedp.Evaluate(`!!document.querySelector('tr:nth-child(3) input[value="0230"]')`, &isExist1),
	); err != nil {
		log.Println("Failed to check if the checkbox is exist")
	}

	if err := chromedp.Run(ctx,
		chromedp.Sleep(1*time.Second),
		chromedp.Evaluate(`!!document.querySelector('tr:nth-child(3) input[value="0231"]')`, &isExist2),
	); err != nil {
		log.Println("Failed to check if the checkbox is exist")
	}

	if err := chromedp.Run(ctx,
		chromedp.Sleep(1*time.Second),
		chromedp.Evaluate(`!!document.querySelector('tr:nth-child(3) input[value="0232"]')`, &isExist3),
	); err != nil {
		log.Println("Failed to check if the checkbox is exist")
	}

	if err := chromedp.Run(ctx,
		chromedp.Sleep(1*time.Second),
		chromedp.Evaluate(`!!document.querySelector('tr:nth-child(3) input[value="0233"]')`, &isExist4),
	); err != nil {
		log.Println("Failed to check if the checkbox is exist")
	}

	if isExist1 && isExist2 && isExist3 && isExist4 {
		fmt.Println("The checkbox is all exist.")
	} else {
		if !isExist1 {
			fmt.Println("【エラー：他のユーザーにより予約済み】20:30 ~ 21:00枠")
		}
		if !isExist2 {
			fmt.Println("【エラー：他のユーザーにより予約済み】21:00 ~ 21:30枠")
		}
		if !isExist3 {
			fmt.Println("【エラー：他のユーザーにより予約済み】21:30 ~ 22:00枠")
		}
		if !isExist4 {
			fmt.Println("【エラー：他のユーザーにより予約済み】22:00 ~ 22:30枠")
		}

		log.Fatal("【エラー：他のユーザーにより予約済み】枠が埋まっているため予約できません。")
	}

	// 時間帯を選択
	if err := chromedp.Run(ctx,
		chromedp.Click(`tr:nth-child(3) input[value="0230"]`, chromedp.ByQuery),
		chromedp.Click(`tr:nth-child(3) input[value="0231"]`, chromedp.ByQuery),
		chromedp.Click(`tr:nth-child(3) input[value="0232"]`, chromedp.ByQuery),
		chromedp.Click(`tr:nth-child(3) input[value="0233"]`, chromedp.ByQuery),
	); err != nil {
		log.Fatal("Failed to check", err)
	}

	// チェックが入っているか確認
	var isChecked1 bool
	var isChecked2 bool
	var isChecked3 bool
	var isChecked4 bool
	if err := chromedp.Run(ctx,
		chromedp.Evaluate(`document.querySelector('tr:nth-child(3) input[value="0230"]').checked`, &isChecked1),
		chromedp.Evaluate(`document.querySelector('tr:nth-child(3) input[value="0231"]').checked`, &isChecked2),
		chromedp.Evaluate(`document.querySelector('tr:nth-child(3) input[value="0232"]').checked`, &isChecked3),
		chromedp.Evaluate(`document.querySelector('tr:nth-child(3) input[value="0233"]').checked`, &isChecked4),
	); err != nil {
		log.Fatal("Failed to check", err)
	}

	if isChecked1 && isChecked2 && isChecked3 && isChecked4 {
		fmt.Println("The input is all checked.")
	} else {
		log.Fatal("The input is not checked.")
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
