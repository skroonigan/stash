package scraper

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/stashapp/stash/pkg/logger"
	"github.com/stashapp/stash/pkg/models"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

func GetPerformerNames(q string) ([]string, error) {
	// Request the HTML page.
	queryURL := "https://www.freeones.com/suggestions.php?q=" + url.PathEscape(q) + "&t=1"
	res, err := http.Get(queryURL)
	if err != nil {
		logger.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("status code error: %d %s", res.StatusCode, res.Status)
	}

	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return nil, err
	}

	// Find the performers
	var performerNames []string
	doc.Find(".suggestion").Each(func(i int, s *goquery.Selection) {
		name := strings.Trim(s.Text(), " ")
		performerNames = append(performerNames, name)
	})

	return performerNames, nil
}

func GetPerformer(performerName string) (*models.ScrapedPerformer, error) {
	queryURL := "https://www.freeones.com/search/?t=1&q=" + url.PathEscape(performerName) + "&view=thumbs"
	res, err := http.Get(queryURL)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("status code error: %d %s", res.StatusCode, res.Status)
	}

	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return nil, err
	}

	performerLink := doc.Find("div.Block3 a").FilterFunction(func(i int, s *goquery.Selection) bool {
		href, _ := s.Attr("href")
		if href == "/html/j_links/Jenna_Leigh_c/" || href == "/html/a_links/Alexa_Grace_c/" {
			return false
		}
		if strings.ToLower(s.Text()) == strings.ToLower(performerName) {
			return true
		}
		alias := s.ParentsFiltered(".babeNameBlock").Find(".babeAlias").First();
		if strings.Contains( strings.ToLower(alias.Text()), strings.ToLower(performerName) ) {
			return true
		}
		return false
	})

	href, _ := performerLink.Attr("href")
	href = strings.TrimSuffix(href, "/")
	regex := regexp.MustCompile(`.+_links\/(.+)`)
	matches := regex.FindStringSubmatch(href)
	if len(matches) < 2 {
		return nil, fmt.Errorf("No matches found in %s",href)
	}

	href = strings.Replace(href, matches[1], "bio_"+matches[1]+".php", -1)
	href = "https://www.freeones.com" + href
	
	bioRes, err := http.Get(href)
	if err != nil {
		return nil, err
	}
	defer bioRes.Body.Close()
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("status code error: %d %s", res.StatusCode, res.Status)
	}

	// Load the HTML document
	bioDoc, err := goquery.NewDocumentFromReader(bioRes.Body)
	if err != nil {
		return nil, err
	}

	params := bioDoc.Find(".paramvalue")
	paramIndexes := getIndexes(bioDoc)

	result := models.ScrapedPerformer{}

	performerURL := bioRes.Request.URL.String()
	result.URL = &performerURL

	name := paramValue(params, paramIndexes["name"])
	result.Name = &name

	ethnicity := getEthnicity(paramValue(params, paramIndexes["ethnicity"]))
	result.Ethnicity = &ethnicity

	country := paramValue(params, paramIndexes["country"])
	result.Country = &country

	eyeColor := paramValue(params, paramIndexes["eye_color"])
	result.EyeColor = &eyeColor

	measurements := paramValue(params, paramIndexes["measurements"])
	result.Measurements = &measurements

	fakeTits := paramValue(params, paramIndexes["fake_tits"])
	result.FakeTits = &fakeTits

	careerLength := paramValue(params, paramIndexes["career_length"])
	careerRegex := regexp.MustCompile(`\([\s\S]*`)
	careerLength = careerRegex.ReplaceAllString(careerLength, "")
	careerLength = trim(careerLength)
	result.CareerLength = &careerLength

	tattoos := paramValue(params, paramIndexes["tattoos"])
	result.Tattoos = &tattoos

	piercings := paramValue(params, paramIndexes["piercings"])
	result.Piercings = &piercings

	aliases := paramValue(params, paramIndexes["aliases"])
	result.Aliases = &aliases

	birthdate := paramValue(params, paramIndexes["birthdate"])
	birthdateRegex := regexp.MustCompile(` \(\d* years old\)`)
	birthdate = birthdateRegex.ReplaceAllString(birthdate, "")
	birthdate = trim(birthdate)
	if birthdate != "Unknown" && len(birthdate) > 0 {
		t, _ := time.Parse("January _2, 2006", birthdate) // TODO
		formattedBirthdate := t.Format("2006-01-02")
		result.Birthdate = &formattedBirthdate
	}

	height := paramValue(params, paramIndexes["height"])
	heightRegex := regexp.MustCompile(`heightcm = "(.*)"\;`)
	heightMatches := heightRegex.FindStringSubmatch(height)
	if len(heightMatches) > 1 {
		result.Height = &heightMatches[1]
	}

	twitterElement := bioDoc.Find(".twitter a")
	twitterHref, _ := twitterElement.Attr("href")
	if twitterHref != "" {
		twitterURL, _ := url.Parse(twitterHref)
		twitterHandle := strings.Replace(twitterURL.Path, "/", "", -1)
		result.Twitter = &twitterHandle
	}

	instaElement := bioDoc.Find(".instagram a")
	instaHref, _ := instaElement.Attr("href")
	if instaHref != "" {
		instaURL, _ := url.Parse(instaHref)
		instaHandle := strings.Replace(instaURL.Path, "/", "", -1)
		result.Instagram = &instaHandle
	}

	return &result, nil
}

func getIndexes(doc *goquery.Document) map[string]int {
	var indexes = make(map[string]int)
	doc.Find(".paramname").Each(func(i int, s *goquery.Selection) {
		index := i + 1
		paramName := trim(s.Text())
		switch paramName {
		case "Babe Name:":
			indexes["name"] = index
		case "Ethnicity:":
			indexes["ethnicity"] = index
		case "Country of Origin:":
			indexes["country"] = index
		case "Date of Birth:":
			indexes["birthdate"] = index
		case "Eye Color:":
			indexes["eye_color"] = index
		case "Height:":
			indexes["height"] = index
		case "Measurements:":
			indexes["measurements"] = index
		case "Fake boobs:":
			indexes["fake_tits"] = index
		case "Career Start And End":
			indexes["career_length"] = index
		case "Tattoos:":
			indexes["tattoos"] = index
		case "Piercings:":
			indexes["piercings"] = index
		case "Aliases:":
			indexes["aliases"] = index
		}
	})
	return indexes
}

func getEthnicity(ethnicity string) string {
	switch ethnicity {
	case "Caucasian":
		return "white"
	case "Black":
		return "black"
	case "Latin":
		return "hispanic"
	case "Asian":
		return "asian"
	default:
		panic("unknown ethnicity")
	}
}

func paramValue(params *goquery.Selection, paramIndex int) string {
	i := paramIndex - 1
	if paramIndex <= 0 {
		return ""
	}
	node := params.Get(i).FirstChild
	content := trim(node.Data)
	if content != "" {
		return content
	}
	node = node.NextSibling
	if (node == nil) {
		return ""
	}
	return trim(node.FirstChild.Data)
}

// https://stackoverflow.com/questions/20305966/why-does-strip-not-remove-the-leading-whitespace
func trim(text string) string {
	// return text.replace(/\A\p{Space}*|\p{Space}*\z/, "");
	return strings.TrimSpace(text)
}
