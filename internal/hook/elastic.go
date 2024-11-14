package hook

import (
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/raphi011/handoff/internal/model"
)

// ElasticSearchHook supports fetching logs created by test runs.
type ElasticSearchHook struct {
	client *elasticsearch.Client

	// searchKeys is a list of runContext keys that can be used
	// to query elasticsearch for relevant logs
	searchKeys []string
}

func (p *ElasticSearchHook) Name() string {
	return "elastic-search"
}

func (p *ElasticSearchHook) Init() error {
	return nil
}

const (
	logstashCorrelationIDKey = "elastic-search.correlationID"
)

func (p *ElasticSearchHook) TestFinished(
	suite model.TestSuite,
	run model.TestSuiteRun,
	testName string,
	runContext map[string]any) {
	p.fetchLogsByCorrelationID(runContext)
}

func (p *ElasticSearchHook) fetchLogsByCorrelationID(runContext map[string]any) {
	// es := p.client

	// query := map[string]string{}

	// for _, k := range p.searchKeys {
	// 	if v, ok := runContext["elastic-search."+k]; ok {
	// 		query[k] = v.
	// 	}
	// }

	// var buf bytes.Buffer
	// query := map[string]interface{}{
	// 	"query": map[string]interface{}{
	// 		"match": map[string]interface{}{
	// 			"title": "test",
	// 		},
	// 	},
	// }
	// if err := json.NewEncoder(&buf).Encode(query); err != nil {
	// 	log.Fatalf("Error encoding query: %s", err)
	// }

	// res, err := es.Search(
	// 	es.Search.WithContext(context.Background()),
	// 	es.Search.WithIndex("test"),
	// 	es.Search.WithBody(&buf),
	// 	es.Search.WithTrackTotalHits(true),
	// 	es.Search.WithPretty(),
	// )
	// if err != nil {
	// 	log.Fatalf("Error getting response: %s", err)
	// }
	// defer res.Body.Close()

	// if res.IsError() {
	// 	var e map[string]interface{}
	// 	if err := json.NewDecoder(res.Body).Decode(&e); err != nil {
	// 		log.Printf("unable to fetch elasticsearch logs: %s", err)
	// 	} else {
	// 		// Print the response status and error information.
	// 		log.Printf("unable to fetch elasticserach logs [%s] %s: %s",
	// 			res.Status(),
	// 			e["error"].(map[string]interface{})["type"],
	// 			e["error"].(map[string]interface{})["reason"],
	// 		)
	// 	}
	// }

	// var r map[string]interface{}

	// if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
	// 	log.Fatalf("Error parsing the response body: %s", err)
	// }
	// // Print the response status, number of results, and request duration.
	// log.Printf(
	// 	"[%s] %d hits; took: %dms",
	// 	res.Status(),
	// 	int(r["hits"].(map[string]interface{})["total"].(map[string]interface{})["value"].(float64)),
	// 	int(r["took"].(float64)),
	// )
	// // Print the ID and document source for each hit.
	// for _, hit := range r["hits"].(map[string]interface{})["hits"].([]interface{}) {
	// 	log.Printf(" * ID=%s, %s", hit.(map[string]interface{})["_id"], hit.(map[string]interface{})["_source"])
	// }

}
