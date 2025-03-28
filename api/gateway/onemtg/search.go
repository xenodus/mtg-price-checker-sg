package onemtg

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strings"

	"github.com/gocolly/colly/v2"
	"mtg-price-checker-sg/gateway"
	"mtg-price-checker-sg/gateway/binderpos"
)

const StoreName = "OneMtg"
const StoreBaseURL = "https://onemtg.com.sg"
const StoreSearchURL = "/search?q=%s"

const binderposStoreURL = "one-mtg.myshopify.com"

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
	return scrap(s, searchStr)
}

func scrap(s Store, searchStr string) ([]gateway.Card, error) {
	searchURL := s.BaseUrl + fmt.Sprintf(s.SearchUrl, url.QueryEscape(searchStr))
	var cards []gateway.Card

	c := colly.NewCollector()

	c.OnHTML("body", func(e *colly.HTMLElement) {
		e.ForEach("div", func(_ int, el *colly.HTMLElement) {
			cardInfoStr := el.Attr("data-product-variants")
			if len(cardInfoStr) > 0 {
				productId := el.Attr("data-product-id")
				var pageUrl, imgUrl string
				if len(productId) > 0 {
					pageUrl = e.ChildAttr("div.product-card-list2__"+productId+" a", "href")
					imgUrl = e.ChildAttr("div.product-card-list2__"+productId+" img", "src")
				}

				var cardInfo []binderpos.CardInfo
				err := json.Unmarshal([]byte(cardInfoStr), &cardInfo)
				if err == nil {
					if len(cardInfo) > 0 && len(pageUrl) > 0 && len(imgUrl) > 0 {
						for _, card := range cardInfo {
							// url with variant (quality)
							u, err := url.Parse(strings.TrimSpace(s.BaseUrl + pageUrl))
							if err != nil {
								log.Printf("error parsing url for %s with value [%s]: %v", s.Name, pageUrl, err)
								return
							}
							q := url.Values{
								"variant": []string{fmt.Sprint(card.ID)},
							}

							cleanPageURL := fmt.Sprintf("%s://%s%s?%s", u.Scheme, u.Host, u.Path, q.Encode())

							cards = append(cards, gateway.Card{
								Name:       strings.TrimSpace(card.Name),
								Url:        strings.TrimSpace(cleanPageURL),
								InStock:    card.Available,
								Price:      float64(card.Price) / 100,
								Source:     s.Name,
								Img:        strings.TrimSpace("https:" + imgUrl),
								Quality:    card.Title,
								IsScrapped: true,
							})
						}
					}
				}
			}
		})
	})

	return cards, c.Visit(searchURL)
}
