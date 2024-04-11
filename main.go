package main

import (
	"context"
	"encoding/csv"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"google.golang.org/api/indexing/v3"
	"google.golang.org/api/option"
)

// Struct to parse sitemap.xml
type Urlset struct {
	XMLName xml.Name `xml:"urlset"`
	Urls    []Url    `xml:"url"`
}

type Url struct {
	Loc string `xml:"loc"`
}

// Struct for Google Index API request body
type IndexRequestBody struct {
	Url  string `json:"url"`
	Type string `json:"type"`
}

// Load environment variables
var (
	credentialsFile = os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	sitemapFile     = os.Getenv("SITEMAP_FILE")
	indexedFile     = os.Getenv("INDEXED_FILE")
	sentFile        = os.Getenv("SENT_FILE")
	rateLimitDay    = os.Getenv("RATE_LIMIT_PER_DAY")
	rateLimitMinute = os.Getenv("RATE_LIMIT_PER_MINUTE")
)

func main() {
	ctx := context.Background()
	client, err := indexing.NewService(ctx, option.WithCredentialsFile(credentialsFile))
	if err != nil {
		log.Fatal("Error creating indexing service:", err)
		return
	}

	rateLimitDayInt, err := strconv.Atoi(rateLimitDay)
	if err != nil {
		log.Fatal("Error converting rate limit per day to integer:", err)
		return
	}

	rateLimitMinuteInt, err := strconv.Atoi(rateLimitMinute)
	if err != nil {
		log.Fatal("Error converting rate limit per minute to integer:", err)
		return
	}

	sleepDur := time.Minute/time.Duration(rateLimitMinuteInt) + time.Millisecond*100

	fmt.Println("Sleep duration (s): ", sleepDur.Seconds())

	// Parse sitemap.xml
	urls, err := parseSitemap(sitemapFile)
	if err != nil {
		log.Fatal("Error parsing sitemap:", err)
		return
	}

	// Read indexed and sent URLs from CSV files
	indexedUrls, err := readCsv(indexedFile)
	if err != nil {
		log.Fatal("Error reading indexed URLs:", err)
		return
	}

	sentUrls, err := readCsv(sentFile)
	if err != nil {
		log.Fatal("Error reading sent URLs:", err)
		return
	}

	todayAlreadySent, err := todaySent(sentFile)
	if err != nil {
		log.Fatal("Error reading today's sent URLs:", err)
		return
	}

	// Correct day limit
	todayLimit := rateLimitDayInt - todayAlreadySent

	fmt.Printf("Today's limit: %d\n", todayLimit)

	count := 0
	// Send URLs to Google Index API
	for _, url := range urls {
		if !contains(indexedUrls, url) && !contains(sentUrls, url) {
			count++
			if count > todayLimit {
				// Sleep for a day
				fmt.Println("Sleeping for a 24 hours...")
				time.Sleep(24 * time.Hour)
				count = 0
				todayLimit = rateLimitDayInt
			}

			fmt.Println("Sending URL to Index API:", url)

			notification := indexing.UrlNotification{
				Type: "URL_UPDATED",
				Url:  url,
			}
			res, err := client.UrlNotifications.Publish(&notification).Do()
			if err != nil {
				fmt.Println("Error sending URL to Index API:", err)
				continue
			}

			// If status is not 200, log the error
			if res.HTTPStatusCode != 200 {
				fmt.Printf("Status code: %d\n", res.HTTPStatusCode)
				continue
			}

			// Append the sent URL to sent.csv
			err = appendUrlToCsv(sentFile, url)
			if err != nil {
				fmt.Println("Error appending URL to sent.csv:", err)
				continue
			}
			time.Sleep(sleepDur)
		}
	}
	fmt.Printf("Finish. Sent %d URLs to Google Index API\n", count)
}

// parseSitemap parses the given sitemap.xml file and returns a slice of URLs
func parseSitemap(filePath string) ([]string, error) {
	xmlFile, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer xmlFile.Close()

	bytes, err := ioutil.ReadAll(xmlFile)
	if err != nil {
		return nil, err
	}

	var urlset Urlset
	err = xml.Unmarshal(bytes, &urlset)
	if err != nil {
		return nil, err
	}

	var urls []string
	for _, url := range urlset.Urls {
		urls = append(urls, url.Loc)
	}

	return urls, nil
}

// readCsv reads URLs from a CSV file and returns them as a map
func readCsv(filePath string) (map[string]struct{}, error) {
	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	csvReader := csv.NewReader(file)
	records, err := csvReader.ReadAll()
	if err != nil {
		return nil, err
	}

	urls := map[string]struct{}{}
	for _, record := range records {
		urls[record[0]] = struct{}{}
	}

	return urls, nil
}

// todaySent reads the number of URLs sent today from a CSV file
func todaySent(filePath string) (int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	csvReader := csv.NewReader(file)
	records, err := csvReader.ReadAll()
	if err != nil {
		return 0, err
	}

	today := time.Now().Format("2006-01-02")
	sent := 0
	// Read date stored in record[1] in a format 2024-04-05 and determine if it is today
	for _, record := range records {
		if strings.HasPrefix(record[1], today) {
			sent++
		}
	}
	return sent, nil
}

// contains checks if a map contains a given string
func contains(m map[string]struct{}, str string) bool {
	_, ok := m[str]
	return ok
}

// appendUrlToCsv appends a URL to a CSV file
func appendUrlToCsv(filePath, url string) error {
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	err = writer.Write([]string{url, time.Now().Format(time.RFC3339)})
	if err != nil {
		return err
	}
	writer.Flush()

	return nil
}
