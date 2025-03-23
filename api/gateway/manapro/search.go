package manapro

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/gocolly/colly/v2"
	"mtg-price-checker-sg/gateway"
	"mtg-price-checker-sg/gateway/binderpos"
)

const StoreName = "Mana Pro"
const StoreBaseURL = "https://sg-manapro.com"
const StoreSearchURL = "/search?type=product&q="

const binderposStoreURL = "mana-pro-sg.myshopify.com"

type Store struct {
	Name      string
	BaseUrl   string
	SearchUrl string
}

func NewLGS() gateway.LGS {
	return Store{
		Name:      StoreName,
		BaseUrl:   StoreBaseURL,
		SearchUrl: StoreSearchURL,
	}
}

func (s Store) Search(searchStr string) ([]gateway.Card, error) {
	reqPayload, err := json.Marshal(binderpos.Payload{
		StoreURL:    binderposStoreURL,
		Game:        binderpos.ProductTypeMTG.ToString(),
		Title:       searchStr,
		InstockOnly: true,
	})
	if err != nil {
		return []gateway.Card{}, err
	}

	cards, httpStatusCode, err := binderpos.GetCards(s.Name, s.BaseUrl, reqPayload)
	if err != nil {
		return cards, err
	}

	if httpStatusCode != http.StatusOK {
		log.Printf("falling back to scrap for [%s]", s.Name)
		return scrap(s, searchStr)
	}

	return cards, nil
}

func scrap(s Store, searchStr string) ([]gateway.Card, error) {
	searchURL := s.BaseUrl + s.SearchUrl + url.QueryEscape(searchStr)
	var cards []gateway.Card

	c := colly.NewCollector()

	c.OnHTML("div.products-display", func(e *colly.HTMLElement) {
		e.ForEach("div.product-card-list2", func(_ int, el *colly.HTMLElement) {
			var (
				isInstock bool
				price     float64
			)

			// in stock
			if len(el.ChildTexts("a.addToCart span.value")) > 0 {
				isInstock = el.ChildTexts("a.addToCart span.value")[len(el.ChildTexts("a.addToCart span.value"))-1] != "SOLD OUT"
			}

			if isInstock {
				el.ForEach("select.product-form__variants[name=\"id\"] option", func(_ int, el2 *colly.HTMLElement) {
					if el2.Attr("data-available") != "0" && el2.Attr("data-price") != "" {
						priceStr := el2.Attr("data-price")

						priceStr = strings.Replace(priceStr, "$", "", -1)
						priceStr = strings.Replace(priceStr, ",", "", -1)
						price, _ = strconv.ParseFloat(strings.TrimSpace(priceStr), 64)

						if price > 0 {
							cards = append(cards, gateway.Card{
								Name:    strings.TrimSpace(el.ChildText("div.grid-view-item__title")),
								Url:     strings.TrimSpace(s.BaseUrl + el.ChildAttr("a", "href")),
								InStock: isInstock,
								Price:   price,
								Source:  s.Name,
								Img:     strings.TrimSpace("https:" + el.ChildAttr("img", "src")),
								Quality: strings.TrimSpace(el2.Text),
							})
						}
					}
				})
			}
		})
	})

	return cards, c.Visit(searchURL)
}
