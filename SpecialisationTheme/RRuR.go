package main

import (
	"fmt"
	"flag"
	"os"
	"net/http"
	"net/url"
	"io"
	"golang.org/x/net/html"
	"log"
	"strings"
	"io/ioutil"
	"github.com/gookit/color"
	"time"
	"strconv"
	"sync"
)

var bannerlogo = `   _____   _____            _____      
  /\  _ \ /\  _ \          /\  _ \
  \ \ \_\ \ \ \_\ \  __  __\ \ \_\ \
   \ \    /\ \    / /\ \/\ \\ \    /
    \ \ \\ \\ \ \\ \\ \ \_\ \\ \ \\ \
     \ \_\ \_\ \_\ \_\ \____/ \ \_\ \_\
      \/_/\/ /\/_/\/ /\/___/   \/_/\/ /
          
           v.1.0`

var wg sync.WaitGroup

var wayback_urls_str string
var wayback_urls []string
var fuzzed_urls []string
var ufuzzed_urls []string
var redvaluesarr []int

var tobreak = false

func cspfinder(domain string) {
	endpoint := "https://csp-evaluator.withgoogle.com/getCSP"
	data := url.Values{}
	data.Set("url", domain)
	client := &http.Client{}
	r, err := http.NewRequest("POST", endpoint, strings.NewReader(data.Encode()))
	if err != nil {
		fmt.Println(err)
	}
	res, err := client.Do(r)
	if err != nil {
		fmt.Println(err)
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
	}
	if (strings.Contains(string(body), "error") == true) {
	        fmt.Println("No CSP header found!")
	} else {
		fmt.Println("CSP headers found!")
		csp_part := strings.Split(string(body), "\"status\": \"ok\", \"csp\": ")
		csp_replace1 := strings.Replace(string(csp_part[1]), "}", "", -1)
		csp_replace2 := strings.Replace(csp_replace1, "\"", "", -1)
		csp_split := strings.Split(csp_replace2, "; ")
		for _,part := range(csp_split) {
			csp_split1 := strings.Split(string(part), " ")
			for round,part2 := range(csp_split1) {
				if (round == 0){
					fmt.Printf("\n%s\n", part2)
				} else {
					fmt.Printf("	%s\n", part2)
				}
			}
		}
	}
}

func unique() {
	occured := map[string]bool{}
	for e := range(fuzzed_urls) {
		if (occured[fuzzed_urls[e]] != true) {
			occured[fuzzed_urls[e]] = true
			ufuzzed_urls = append(ufuzzed_urls, fuzzed_urls[e])
		}
	}
}

func fuzzit() {
	for _,uri := range(wayback_urls) {
		uri_replace := strings.Replace(string(uri), "&", "=", -1)
		uri_split := strings.Split(uri_replace, "=")
		var uri_parts = ""
		for round,part := range(uri_split) {
			if (round == 0) {
				uri_parts = uri_parts + part
			} else if (round % 2 == 0) {
				uri_parts = uri_parts + "&" + part
			} else {
				uri_parts = uri_parts + "=FUZZ"
			}
		}
		fuzz := "FUZZ"
		if (strings.Contains(uri_parts, fuzz) == true) {
			fuzzed_urls = append(fuzzed_urls, uri_parts)
		}
	}
	unique()
}

func waybackurls(domain string) {
	fullurl := "https://web.archive.org/cdx/search/cdx?url=" + domain + "/*&fl=original&collapse=urlkey"
	response, err := http.Get(fullurl)
	if (err != nil) {
		fmt.Println("Error parsing URLs")
		os.Exit(0)
	}
	defer response.Body.Close()
	tokenizer := html.NewTokenizer(response.Body)
	for {
		tt := tokenizer.Next()
		t := tokenizer.Token()
		err := tokenizer.Err()
		if (err == io.EOF) {
			break
		}
		switch tt {
		case html.ErrorToken:
			log.Fatal(err)
		case html.TextToken:
			data := strings.TrimSpace(t.Data)
			wayback_urls_str = wayback_urls_str + data
		}
	}
	wayback_urls = strings.Split(wayback_urls_str, "\n")
}

func getfuzzfuzz(urls []string, sleep int, cookie_name string, cookie_value string) {
	for _,url := range(urls) {
		req, err := http.NewRequest("GET", url, nil)
		if (err != nil) {
			fmt.Println("Error with making a request")
		}

		cookie := http.Cookie{Name: cookie_name, Value: cookie_value}
		req.AddCookie(&cookie)

		client := &http.Client{}
		resp, err := client.Do(req)
		if (err != nil) {
			fmt.Println("Error: couldn't get a response")
		}
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if (err != nil) {
			fmt.Println("Error: cannot get response body")
		}

		if (strings.Contains(string(body), "FUZZ") == true) {
			color.Cyan.Printf("%v	%v	  %s\n", resp.StatusCode, len(string(body)), url)
		} else {
			fmt.Printf("%v	%v	  %s\n", resp.StatusCode, len(string(body)), url)
		}
		time.Sleep(time.Duration(sleep) * time.Millisecond)
	}
	wg.Done()
}

func getxssfuzz(urls []string, payloads []string, sleep int, cookie_name string, cookie_value string) {
	for _,url := range(urls) {
		for _,payload := range(payloads) {
			fuzzurl := strings.Replace(url, "FUZZ", payload, -1)
			req, err := http.NewRequest("GET", fuzzurl, nil)
			if (err != nil) {
				fmt.Println("Error with making a request")
			}

			cookie := http.Cookie{Name: cookie_name, Value: cookie_value}
			req.AddCookie(&cookie)

			client := &http.Client{}
			resp, err := client.Do(req)
			if (err != nil) {
				fmt.Println("Error: couldn't get a response")
			}
			defer resp.Body.Close()

			body, err := ioutil.ReadAll(resp.Body)
			if (err != nil) {
				fmt.Println("Error: cannot get response body")
			}

			if (strings.Contains(string(body), "alert(5231)") == true) {
				color.Cyan.Printf("%v	%v	  %s\n", resp.StatusCode, len(string(body)), fuzzurl)
				if (tobreak == false) {
					break
				}
			} else {
				fmt.Printf("%v	%v	  %s\n", resp.StatusCode, len(string(body)), fuzzurl)
			}

			time.Sleep(time.Duration(sleep) * time.Millisecond)
		}
	}
	wg.Done()
}

func getsqlfuzz(urls []string, payloads []string, sleep int, cookie_name string, cookie_value string) {
	for _,url := range(urls) {
		for _,payload := range(payloads) {
			fuzzurl := strings.Replace(url, "FUZZ", payload, -1)
			req, err := http.NewRequest("GET", url, nil)
			if (err != nil) {
				fmt.Println("Error with making a request")
			}

			cookie := http.Cookie{Name: cookie_name, Value: cookie_value}
			req.AddCookie(&cookie)

			client := &http.Client{}
			resp, err := client.Do(req)
			if (err != nil) {
				fmt.Println("Error: couldn't get a response")
			}
			defer resp.Body.Close()

			body, err := ioutil.ReadAll(resp.Body)
			if (err != nil) {
				fmt.Println("Error: cannot get response body")
			}
			if (strings.Contains(strings.ToLower(string(body)), "sql error") == true || strings.Contains(strings.ToLower(string(body)), "sql syntax") == true || strings.Contains(strings.ToLower(string(body)), "unexpected end of command in statement") == true || strings.Contains(strings.ToLower(string(body)), "mysql_fetch") == true) {
				color.Cyan.Printf("%v	%v	  %s\n", resp.StatusCode, len(string(body)), fuzzurl)
				if (tobreak == false) {
					break
				}
			} else {
				fmt.Printf("%v	%v	  %s\n", resp.StatusCode, len(string(body)), fuzzurl)
			}

			time.Sleep(time.Duration(sleep) * time.Millisecond)
		}
	}
	wg.Done()
}

func getlfifuzz(urls []string, payloads []string, sleep int, cookie_name string, cookie_value string) {
	for _,url := range(urls) {
		for _,payload := range(payloads) {
			fuzzurl := strings.Replace(url, "FUZZ", payload, -1)
			req, err := http.NewRequest("GET", fuzzurl, nil)
			if (err != nil) {
				fmt.Println("Error with making a request")
			}

			cookie := http.Cookie{Name: cookie_name, Value: cookie_value}
			req.AddCookie(&cookie)

			client := &http.Client{}
			resp, err := client.Do(req)
			if (err != nil) {
				fmt.Println("Error: couldn't get a response")
			}
			defer resp.Body.Close()

			body, err := ioutil.ReadAll(resp.Body)
			if (err != nil) {
				fmt.Println("Error: cannot get response body")
			}
			if (strings.Contains(string(body), "root:x:") == true || strings.Contains(string(body), "root:$") == true) {
				color.Cyan.Printf("%v	%v	  %s\n", resp.StatusCode, len(string(body)), fuzzurl)
				if (tobreak == false) {
					break
				}
			} else {
				fmt.Printf("%v	%v	  %s\n", resp.StatusCode, len(string(body)), fuzzurl)
			}

			time.Sleep(time.Duration(sleep) * time.Millisecond)
		}
	}
	wg.Done()
}

func checkwaf(urls []string, payloads []string, sleep int, cookie_name string, cookie_value string) {
	for _,url := range(urls) {
		for _,payload := range(payloads) {
			fuzzurl := strings.Replace(url, "FUZZ", payload, -1)
			req, err := http.NewRequest("GET", fuzzurl, nil)
			if (err != nil) {
				fmt.Println("Error with making a request")
			}

			cookie := http.Cookie{Name: cookie_name, Value: cookie_value}
			req.AddCookie(&cookie)

			client := &http.Client{}
			resp, err := client.Do(req)
			if (err != nil) {
				fmt.Println("Error: couldn't get a response")
			}
			defer resp.Body.Close()

			body, err := ioutil.ReadAll(resp.Body)
			if (err != nil) {
				fmt.Println("Error: cannot get response body")
			}

			len_body := len(string(body))
			var checkred = 0
			if (len(redvaluesarr) > 0) {
				for _,value := range(redvaluesarr) {
					if (len_body == value) {
						color.Magenta.Printf("%v	%v	  %s		%s\n", resp.StatusCode, len_body, payload, fuzzurl)
						checkred = 1
						break
					}
				}
				if (checkred == 0) {
					fmt.Printf("%v	%v	  %s		%s\n", resp.StatusCode, len_body, payload, fuzzurl)
				}
			} else {
				fmt.Printf("%v	%v	  %s		%s\n", resp.StatusCode, len_body, payload, fuzzurl)
			}
			time.Sleep(time.Duration(sleep) * time.Millisecond)
		}
	}
	wg.Done()
}

func main() {
	var domain string
	flag.StringVar(&domain, "d", "-", "Target URL")
	var file string
	flag.StringVar(&file, "f", "-", "File containing URLs")
	var url string
	flag.StringVar(&url, "u", "-", "URL to use")
	var sleep int
	flag.IntVar(&sleep, "s", 0, "Delay after every request (milliseconds)")
	var threads int
	flag.IntVar(&threads, "th", 8, "Amount of threads to use concurrently")
	fuzzfuzz := flag.Bool("fuzzfuzz", false, "See if the word 'FUZZ' gets reflected")
	var fuzzxss string
	flag.StringVar(&fuzzxss, "fuzzxss", "-", "File to use for XSS fuzzing")
	var fuzzsql string
	flag.StringVar(&fuzzsql, "fuzzsql", "-", "File to use for SQL fuzzing")
	var fuzzlfi string
	flag.StringVar(&fuzzlfi, "fuzzlfi", "-", "File to use for LFI fuzzing")
	var redvalues string
	flag.StringVar(&redvalues, "rv", "-", "Values to highlight red while scanning the WAF")
	var wafxss string
	flag.StringVar(&wafxss, "wafxss", "-", "File to use for WAF XSS testing")
	var wafsql string
	flag.StringVar(&wafsql, "wafsql", "-", "File to use for WAF SQL testing")
	var cookie string
	flag.StringVar(&cookie, "cookie", ":", "Use cookie while making HTTP request (usage: --cookie name:value)")

	fuzzall := flag.Bool("fuzzall", false, "Fuzz for XSS, SQL and LFI")
	find_urls := flag.Bool("findurls", false, "Find urls")
	find_fuzzed_urls := flag.Bool("findfuzzedurls", false, "Find urls and settle fuzz parameters")
	csp := flag.Bool("csp", false, "Check if your target has a CSP")
	nobanner := flag.Bool("nobanner", false, "Don't show the banner")
	nobreak := flag.Bool("nobreak", false, "Don't stop with fuzzing")
	notime := flag.Bool("notime", false, "Don't show how long it took to finish")
	nononsense := flag.Bool("nononsense", false, "notime is true and nobanner is true")
	flag.Parse()

	starttime := time.Now()

	var check_xssfuzz = 0
	var check_sqlfuzz = 0
	var check_lfifuzz = 0

	var xss_payloads []string
	var sql_payloads []string
	var lfi_payloads []string

	if (*nononsense == true) {
		*nobanner = true
		*notime = true
	}

	if (*nobanner == false) {
		fmt.Println(bannerlogo)
		fmt.Printf("___________________________________________________\n\n")
		fmt.Printf(" :: Csp		  : %t\n", *csp)
		fmt.Printf(" :: Domain	  : %s\n", domain)
		fmt.Printf(" :: File	  : %s\n", file)
		fmt.Printf(" :: Url		  : %s\n", url)
		fmt.Printf(" :: Threads	  : %v\n", threads)
		fmt.Printf(" :: Sleep	  : %v\n", sleep)
		if (cookie == ":") {
			fmt.Printf(" :: Cookie	  : -\n")
		} else {
			fmt.Printf(" :: Cookie	  : %s\n", cookie)
		}
		fmt.Printf(" :: Findfurls	  : %t\n", *find_fuzzed_urls)
		fmt.Printf(" :: Findurls	  : %t\n", *find_urls)
		fmt.Printf(" :: Fuzzfuzz	  : %t\n", *fuzzfuzz)
		fmt.Printf(" :: Fuzzsql	  : %s\n", fuzzsql)
		fmt.Printf(" :: Fuzzxss	  : %s\n", fuzzxss)
		fmt.Printf(" :: Fuzzlfi	  : %s\n", fuzzlfi)
		fmt.Printf(" :: Wafxss	  : %s\n", wafxss)
		fmt.Printf(" :: Wafsql	  : %s\n", wafsql)
		fmt.Printf("___________________________________________________\n")

	}

	if (*nobreak == true) {
		tobreak = true
	}

	if (*csp == true) {
		if (domain == "-") {
			fmt.Println("You need to specify a domain")
			os.Exit(0)
		} else {
			cspfinder(domain)
		}
	}

	if (*find_urls == true) {
		if (domain == "-") {
			fmt.Println("You need to specify a domain")
			os.Exit(0)
		} else {
			waybackurls(domain)
			for _,single_url := range(wayback_urls) {
				fmt.Println(single_url)
			}
		}
	}

	if (*find_fuzzed_urls == true) {
		if (domain == "-") {
			fmt.Println("You need to specify a domain")
			os.Exit(0)
		} else {
			waybackurls(domain)
			fuzzit()
			for _,single_url := range(ufuzzed_urls) {
				fmt.Println(single_url)
			}
		}
	}

	if (file != "-") {
		fileread, err := ioutil.ReadFile(file)
		if (err != nil) {
			fmt.Println("Error while reading file")
			os.Exit(0)
		} else {
			filesplit := strings.Split(string(fileread), "\n")
			wayback_urls = filesplit[0:len(filesplit) -1]
			fuzzit()
		}
	}

	if (url != "-") {
		wayback_urls = append(wayback_urls, url)
		fuzzit()
	}

	cookie_split := strings.Split(cookie, ":")
	cookie_name := cookie_split[0]
	cookie_value := cookie_split[1]

	CurrentDirectory, err := os.Getwd()
	if (err != nil) {
		fmt.Println("Error: Can't get working directory")
	}
	CurrentDirectory = CurrentDirectory + "/"

	if (fuzzsql == "-") {
		fmt.Printf("")
	} else if (fuzzsql == "default") {
		check_sqlfuzz = 1
		sql_payloadfile, err := ioutil.ReadFile(CurrentDirectory + "defaultsql.txt")
		if (err != nil) {
			fmt.Println("Error reading the file defaultsql.txt")
			os.Exit(0)
		}
		filesplit := strings.Split(string(sql_payloadfile), "\n")
		sql_payloads = filesplit[0:len(filesplit) -1]
	} else {
		check_sqlfuzz = 1
		sql_payloadfile, err := ioutil.ReadFile(fuzzsql)
		if (err != nil) {
			fmt.Println("Error reading the file defaultsql.txt")
			os.Exit(0)
		}
		filesplit := strings.Split(string(sql_payloadfile), "\n")
		sql_payloads = filesplit[0:len(filesplit) -1]
	}

	if (fuzzxss == "-") {
		fmt.Printf("")
	} else if (fuzzxss == "default") {
		check_xssfuzz = 1
		xss_payloadfile, err := ioutil.ReadFile(CurrentDirectory + "defaultxss.txt")
		if (err != nil) {
			fmt.Println("Error reading the file defaultxss.txt")
			os.Exit(0)
		}
		filesplit := strings.Split(string(xss_payloadfile), "\n")
		xss_payloads = filesplit[0:len(filesplit) -1]
	} else {
		check_xssfuzz = 1
		xss_payloadfile, err := ioutil.ReadFile(fuzzxss)
		if (err != nil) {
			fmt.Println("Error reading the file defaultxss.txt")
			os.Exit(0)
		}
		filesplit := strings.Split(string(xss_payloadfile), "\n")
		xss_payloads = filesplit[0:len(filesplit) -1]
	}

	if (fuzzlfi == "-") {
		fmt.Printf("")
	} else if (fuzzlfi == "default") {
		check_lfifuzz = 1
		lfi_payloadfile, err := ioutil.ReadFile(CurrentDirectory + "defaultlfi.txt")
		if (err != nil) {
			fmt.Println("Error reading the file defaultlfi.txt")
			os.Exit(0)
		}
		filesplit := strings.Split(string(lfi_payloadfile), "\n")
		lfi_payloads = filesplit[0:len(filesplit) -1]
	} else {
		check_lfifuzz = 1
		lfi_payloadfile, err := ioutil.ReadFile(fuzzlfi)
		if (err != nil) {
			fmt.Println("Error reading the file defaultlfi.txt")
			os.Exit(0)
		}
		filesplit := strings.Split(string(lfi_payloadfile), "\n")
		lfi_payloads = filesplit[0:len(filesplit) -1]
	}

	if (*fuzzall == true) {
		check_xssfuzz = 1
		check_sqlfuzz = 1
		check_lfifuzz = 1

		sql_payloadfile, err := ioutil.ReadFile(CurrentDirectory + "defaultsql.txt")
		if (err != nil) {
			fmt.Println("Error reading the file defaultsql.txt")
			os.Exit(0)
		}
		filesplit := strings.Split(string(sql_payloadfile), "\n")
		sql_payloads = filesplit[0:len(filesplit) -1]

		xss_payloadfile, err := ioutil.ReadFile(CurrentDirectory + "defaultxss.txt")
		if (err != nil) {
			fmt.Println("Error reading the file defaultxss.txt")
			os.Exit(0)
		}
		filesplit2 := strings.Split(string(xss_payloadfile), "\n")
		xss_payloads = filesplit2[0:len(filesplit2) -1]

		lfi_payloadfile, err := ioutil.ReadFile(CurrentDirectory + "defaultlfi.txt")
		if (err != nil) {
			fmt.Println("Error reading the file defaultlfi.txt")
			os.Exit(0)
		}
		filesplit3 := strings.Split(string(lfi_payloadfile), "\n")
		lfi_payloads = filesplit3[0:len(filesplit3) -1]
	}

	if (*fuzzfuzz == true) {
		fmt.Println("\nLooking for FUZZ reflections...\n___________________________________________________\n")
		fmt.Println("Status:	Size:	  URL:")
		for thread := 0; thread < threads; thread++ {
			var thread_sites []string
			for singlesite := thread; singlesite < len(ufuzzed_urls); singlesite = singlesite + threads {
				thread_sites = append(thread_sites, ufuzzed_urls[singlesite])
			}
			wg.Add(1)
			go getfuzzfuzz(thread_sites, sleep, cookie_name, cookie_value)
		}
		wg.Wait()
	}

	if (check_xssfuzz == 1) {
		fmt.Println("\nStarting with XSS...\n___________________________________________________\n")
		fmt.Println("Status:	Size:	  URL:")
		for thread := 0; thread < threads; thread++ {
			var thread_sites []string
			for singlesite := thread; singlesite < len(ufuzzed_urls); singlesite = singlesite + threads {
				thread_sites = append(thread_sites, ufuzzed_urls[singlesite])
			}
			wg.Add(1)
			go getxssfuzz(thread_sites, xss_payloads, sleep, cookie_name, cookie_value)
		}
		wg.Wait()
	}

	if (check_sqlfuzz == 1) {
		fmt.Println("\nStarting with SQLi...\n___________________________________________________\n")
		fmt.Println("Status:	Size:	  URL:")
		for thread := 0; thread < threads; thread++ {
			var thread_sites []string
			for singlesite := thread; singlesite < len(ufuzzed_urls); singlesite = singlesite + threads {
				thread_sites = append(thread_sites, ufuzzed_urls[singlesite])
			}
			wg.Add(1)
			go getsqlfuzz(thread_sites, sql_payloads, sleep, cookie_name, cookie_value)
		}
		wg.Wait()
	}

	if (check_lfifuzz == 1) {
		fmt.Println("\nStarting with LFI...\n___________________________________________________\n")
		fmt.Println("Status:	Size:	  URL:")
		for thread := 0; thread < threads; thread++ {
			var thread_sites []string
			for singlesite := thread; singlesite < len(ufuzzed_urls); singlesite = singlesite + threads {
				thread_sites = append(thread_sites, ufuzzed_urls[singlesite])
			}
			wg.Add(1)
			go getlfifuzz(thread_sites, lfi_payloads, sleep, cookie_name, cookie_value)
		}
		wg.Wait()
	}

	if (redvalues != "-") {
		redvalues_split := strings.Split(redvalues, ",")
		for _,values := range(redvalues_split) {
			if (strings.Contains(values, "-") == true) {
				split2 := strings.Split(values, "-")
				minvalue, err := strconv.Atoi(split2[0])
				if (err != nil) {
					fmt.Println("Error: Cannot convert to integer")
					os.Exit(0)
				}
				maxvalue, err := strconv.Atoi(split2[1])
				if (err != nil) {
					fmt.Println("Error: Cannot convert to integer")
					os.Exit(0)
				}
				for i := minvalue; i <= maxvalue; i++ {
					redvaluesarr = append(redvaluesarr, i)
				}
			} else {
				values_int, err := strconv.Atoi(values)
				if (err != nil) {
					fmt.Println("Error: Cannot convert to integer")
					os.Exit(0)
				} else {
					redvaluesarr = append(redvaluesarr, values_int)
				}
			}
		}
	}

	if (wafxss != "-") {
		if (wafxss == "default") {
			wafxssfile, err := ioutil.ReadFile("wafxss.txt")
			if (err != nil) {
				fmt.Println("Error: cannot obtain wafxss.txt")
				os.Exit(0)
			}
			filesplit := strings.Split(string(wafxssfile), "\n")
			wafxsspayloads := filesplit[0:len(filesplit) -1]
			fmt.Println("\nStarting with WAF XSS payloads...\n___________________________________________________\n")
			fmt.Println("Status:	Size:	  payload:		URL:")
			for thread := 0; thread < threads; thread++ {
				var thread_sites []string
				for singlesite := thread; singlesite < len(ufuzzed_urls); singlesite = singlesite + threads {
					thread_sites = append(thread_sites, ufuzzed_urls[singlesite])
				}
				wg.Add(1)
				go checkwaf(thread_sites, wafxsspayloads, sleep, cookie_name, cookie_value)
			}
			wg.Wait()
		} else {
			wafxssfile, err := ioutil.ReadFile(wafxss)
			if (err != nil) {
				fmt.Println("Error: cannot obtain " + wafxss)
				os.Exit(0)
			}
			filesplit := strings.Split(string(wafxssfile), "\n")
			wafxsspayloads := filesplit[0:len(filesplit) -1]
			fmt.Println("\nStarting with WAF XSS payloads...\n___________________________________________________\n")
			fmt.Println("Status:	Size:	  payload:		URL:")
			for thread := 0; thread < threads; thread++ {
				var thread_sites []string
				for singlesite := thread; singlesite < len(ufuzzed_urls); singlesite = singlesite + threads {
					thread_sites = append(thread_sites, ufuzzed_urls[singlesite])
				}
				wg.Add(1)
				go checkwaf(thread_sites, wafxsspayloads, sleep, cookie_name, cookie_value)
			}
			wg.Wait()
		}
	}

	if (wafsql != "-") {
		if (wafsql == "default") {
			wafsqlfile, err := ioutil.ReadFile("wafsql.txt")
			if (err != nil) {
				fmt.Println("Error: cannot obtain wafsql.txt")
				os.Exit(0)
			}

			filesplit := strings.Split(string(wafsqlfile), "\n")
			wafsqlpayloads := filesplit[0:len(filesplit) -1]
			fmt.Println("\nStarting with WAF SQL payloads...\n___________________________________________________\n")
			fmt.Println("Status:	Size:	  payload:		URL:")
			for thread := 0; thread < threads; thread++ {
				var thread_sites []string
				for singlesite := thread; singlesite < len(ufuzzed_urls); singlesite = singlesite + threads {
					thread_sites = append(thread_sites, ufuzzed_urls[singlesite])
				}
				wg.Add(1)
				go checkwaf(thread_sites, wafsqlpayloads, sleep, cookie_name, cookie_value)
			}
			wg.Wait()
		} else {
			wafsqlfile, err := ioutil.ReadFile(wafsql)
			if (err != nil) {
				fmt.Println("Error: cannot obtain " + wafsql)
				os.Exit(0)
			}
			filesplit := strings.Split(string(wafsqlfile), "\n")
			wafsqlpayloads := filesplit[0:len(filesplit) -1]
			fmt.Println("\nStarting with WAF SQL payloads...\n___________________________________________________\n")
			fmt.Println("Status:	Size:	  payload:		URL:")
			for thread := 0; thread < threads; thread++ {
				var thread_sites []string
				for singlesite := thread; singlesite < len(ufuzzed_urls); singlesite = singlesite + threads {
					thread_sites = append(thread_sites, ufuzzed_urls[singlesite])
				}
				wg.Add(1)
				go checkwaf(thread_sites, wafsqlpayloads, sleep, cookie_name, cookie_value)
			}
			wg.Wait()
		}
	}

	if (*notime == false) {
		fmt.Printf("\nCode finished in %v\n", time.Since(starttime))
	}
}
