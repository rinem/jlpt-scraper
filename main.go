package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/gocolly/colly"
)

type Example struct {
	Id       string `json:"id"`
	Sentence string `json:"sentence"`
	Reading  string `json:"reading"`
	Meaning  string `json:"meaning"`
}

type Note struct {
	Id       string    `json:"id"`
	Url      string    `json:"url"`
	Grammar  string    `json:"grammar"`
	Reading  string    `json:"reading"`
	Meaning  string    `json:"meaning"`
	Image    string    `json:"image"`
	Examples []Example `json:"examples"`
}

func main() {
	level := flag.String("level", "N2", "JLPT level (N1, N2, N3, N4, N5)")
	fileType := flag.String("filetype", "csv", "File type to save (csv or json)")
	flag.Parse()

	// Mapping of JLPT level to the number of pages to scrape present in jlpt-sensei
	levelPages := map[string]int{
		"N1": 7,
		"N2": 5,
		"N3": 5,
		"N4": 4,
		"N5": 3,
	}

	pages, ok := levelPages[*level]
	if !ok {
		log.Fatalf("Invalid JLPT level: %s", *level)
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	allNotes := make([]Note, 0)

	collector := colly.NewCollector(
		colly.AllowedDomains("jlptsensei.com", "www.jlptsensei.com"),
	)

	collector.OnHTML("tbody tr.jl-row", func(element *colly.HTMLElement) {
		id := element.ChildText("td.jl-td-num")
		grammar := element.ChildText("td.jl-td-gj a.jl-link")
		reading := element.ChildText("td.jl-td-gr a.jl-link")
		url := element.ChildAttr("a.jl-link", "href")
		meaning := element.ChildText("td.jl-td-gm")

		var image string
		var examples []Example

		noteCollector := collector.Clone()

		noteCollector.OnHTML("#main-content", func(e *colly.HTMLElement) {
			image = e.ChildAttr("#header-image", "src")

			exampleCount := 0
			e.ForEach("div.example-cont", func(_ int, exElement *colly.HTMLElement) {
				if exampleCount < 3 {
					exID := exElement.Attr("id")
					if exID != "" {
						sentence := exElement.ChildText(".example-main p.jp")
						reading := exElement.ChildText(".collapse#" + exID + "_ja .alert-success")
						meaning := exElement.ChildText(".collapse#" + exID + "_en .alert-primary")
						example := Example{
							Id:       exID,
							Sentence: sentence,
							Reading:  reading,
							Meaning:  meaning,
						}
						examples = append(examples, example)
						exampleCount++
					}
				}
			})
		})

		noteCollector.OnRequest(func(request *colly.Request) {
			fmt.Println("Visiting", request.URL.String())
		})

		wg.Add(1)
		go func(url, id, grammar, reading, meaning string) {
			defer wg.Done()
			noteCollector.Visit(url)
			note := Note{
				Id:       id,
				Grammar:  grammar,
				Url:      url,
				Reading:  reading,
				Meaning:  meaning,
				Image:    image,
				Examples: examples,
			}

			mu.Lock()
			allNotes = append(allNotes, note)
			mu.Unlock()
		}(url, id, grammar, reading, meaning)
	})

	for i := 1; i <= pages; i++ {
		url := fmt.Sprintf("https://jlptsensei.com/jlpt-%s-grammar-list/page/%d/", *level, i)
		collector.Visit(url)
	}

	wg.Wait()

	switch strings.ToLower(*fileType) {
	case "csv":
		writeCSV(allNotes, *level)
	case "json":
		writeJSON(allNotes, *level)
	default:
		log.Fatalf("Invalid file type: %s. Only 'csv' or 'json' are supported.", *fileType)
	}
}

func writeJSON(data []Note, level string) {
	filename := fmt.Sprintf("jlptnotes_%s.json", level)
	file, err := os.Create(filename)
	if err != nil {
		log.Println("Unable to create the JSON file:", err)
		return
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", " ")
	if err := encoder.Encode(data); err != nil {
		log.Println("Unable to encode JSON:", err)
		return
	}
	fmt.Println("Data written to", filename)
}

func writeCSV(data []Note, level string) {
	filename := fmt.Sprintf("jlptnotes_%s.csv", level)
	file, err := os.Create(filename)
	if err != nil {
		log.Println("Unable to create the CSV file:", err)
		return
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	headers := []string{"Id", "Url", "Grammar", "Reading", "Meaning", "Image",
		"Example1 ID", "Example1 Sentence", "Example1 Reading", "Example1 Meaning",
		"Example2 ID", "Example2 Sentence", "Example2 Reading", "Example2 Meaning",
		"Example3 ID", "Example3 Sentence", "Example3 Reading", "Example3 Meaning",
	}
	if err := writer.Write(headers); err != nil {
		log.Println("Error writing header to CSV:", err)
		return
	}

	// Write data
	for _, note := range data {
		examples := make([]string, 12)
		for i, ex := range note.Examples {
			idx := i * 4
			examples[idx] = ex.Id
			examples[idx+1] = ex.Sentence
			examples[idx+2] = ex.Reading
			examples[idx+3] = ex.Meaning
		}
		record := []string{note.Id, note.Url, note.Grammar, note.Reading, note.Meaning, note.Image}
		record = append(record, examples...)
		if err := writer.Write(record); err != nil {
			log.Println("Error writing record to CSV:", err)
		}
	}

	fmt.Println("Data written to", filename)
}
