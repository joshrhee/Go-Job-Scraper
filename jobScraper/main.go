package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

var baseURL string = "https://kr.indeed.com/jobs?q=python&limit=50"

type extractedJob struct {
	id       string
	title    string
	location string
	salary   string
	summary  string
}

func main() {
	var jobs []extractedJob
	mainChannel := make(chan []extractedJob)
	totalPages := getPages()

	for page := 0; page < totalPages; page++ {
		go getPage(page, mainChannel)
	}
	for page := 0; page < totalPages; page++ {
		extractedJobs := <-mainChannel

		//extractedJobs... merges multiple arrays
		//If we just do extractedJobs, it will add array inside of array
		jobs = append(jobs, extractedJobs...)
	}
	writeJobs(jobs)
	fmt.Println("Done, extracted ", len(jobs), " jobs")
}

//This function return all jobs in that specific page
//first page url is https://kr.indeed.com/jobs?q=python&limit=50
//second page url is https://kr.indeed.com/jobs?q=python&limit=100
//start is increasing 50 for next page
func getPage(page int, mainChannel chan<- []extractedJob) {
	var jobs []extractedJob
	channel := make(chan extractedJob)

	pageURL := baseURL + "&start=" + strconv.Itoa(page*50)
	fmt.Println("Requesting ", pageURL)
	res, err := http.Get(pageURL)
	checkErr(err)
	checkCode(res)

	//Prevent memory leaks
	defer res.Body.Close()

	//Using a Goquery for scraping
	doc, err := goquery.NewDocumentFromReader(res.Body)
	checkErr(err)

	//Job cards of each page has .jobsearch-SerpJobCard class name
	searchCards := doc.Find(".jobsearch-SerpJobCard")

	searchCards.Each(func(i int, card *goquery.Selection) {
		go extractJob(card, channel)
	})

	for i := 0; i < searchCards.Length(); i++ {
		job := <-channel
		jobs = append(jobs, job)
	}

	mainChannel <- jobs
}

//Return job information of the specific job card
//Return extractedJob struct
func extractJob(card *goquery.Selection, channel chan<- extractedJob) {
	id, _ := card.Attr("data-jk")
	title := cleanString(card.Find(".title>a").Text())
	location := cleanString(card.Find(".sjcl").Text())
	salary := cleanString(card.Find(".salaryText").Text())
	summary := cleanString(card.Find(".summary").Text())
	channel <- extractedJob{
		id:       id,
		title:    title,
		location: location,
		salary:   salary,
		summary:  summary}
}

//return a string without spaces
func cleanString(str string) string {
	//TrimSpace trim(delete) each sides spaces
	//And Fields() put all texts into array for delete all other spaces
	//And Join() join all array's texts
	textInArray := strings.Fields(strings.TrimSpace(str))
	return strings.Join(textInArray, " ")
}

//Getting total page numbers
func getPages() int {
	pages := 0
	res, err := http.Get(baseURL)
	checkErr(err)
	checkCode(res)

	//Prevent memory leaks
	defer res.Body.Close()

	//Using a Goquery for scraping
	doc, err := goquery.NewDocumentFromReader(res.Body)
	checkErr(err)

	//Page list html class name is pagination
	doc.Find(".pagination").Each(func(i int, selection *goquery.Selection) {
		pages = selection.Find("a").Length()
	})

	return pages
}

//Write job informations to the file
func writeJobs(jobs []extractedJob) {
	//Create "jobs.csv" file
	file, err := os.Create("jobs.csv")
	checkErr(err)

	//Create a writer for the file
	w := csv.NewWriter(file)
	//Write data to the file when this function finish
	defer w.Flush()

	header := []string{"Link", "Title", "Location", "Salary", "Summary"}

	//Add header to the writer
	wErr := w.Write(header)
	checkErr(wErr)

	//Add job infos to the writer
	for _, job := range jobs {
		jobSlice := []string{"https://kr.indeed.com/viewjob?jk=" + job.id, job.title, job.location, job.salary, job.summary}
		jobWErr := w.Write(jobSlice)
		checkErr(jobWErr)
	}
}

func checkErr(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

func checkCode(res *http.Response) {
	if res.StatusCode != 200 {
		log.Fatalln("Request failed with Status: ", res.StatusCode)
	}
}
