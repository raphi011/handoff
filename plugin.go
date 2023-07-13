package handoff

import (
	"github.com/elastic/go-elasticsearch/v8"
)

type Plugin interface {
	Name() string
	Init() error
	Stop() error
}

type TestStartedListener interface {
	TestStarted(suite TestSuite, run TestRun, testName string)
}

type TestFinishedListener interface {
	TestFinished(suite TestSuite, run TestRun, testName string, context map[string]any)
}

type TestSuiteStartedListener interface {
	TestSuiteStarted()
}

type TestSuiteFinishedListener interface {
	TestSuiteFinished()
}

// PagerDutyPlugin supports creating and resolving incidents when
// testsuites fail.
type PagerDutyPlugin struct {
}

// GithubPlugin supports running testsuites on PRs.
type GithubPlugin struct {
}

// SlackPlugin supports sending messages to slack channels that inform on
// test runs.
type SlackPlugin struct {
}

// ElasticSearchPlugin supports fetching logs created by test runs.
type ElasticSearchPlugin struct {
	client *elasticsearch.Client

	// searchKeys is a list of runContext keys that can be used
	// to query elasticsearch for relevant logs
	searchKeys []string
}

func (p *ElasticSearchPlugin) Name() string {
	return "elastic-search"
}

func (p *ElasticSearchPlugin) Init() error {
	return nil
}

func (p *ElasticSearchPlugin) Stop() error {
	return nil
}

const (
	logstashCorrelationIDKey = "elastic-search.correlationID"
)

func (p *ElasticSearchPlugin) TestFinished(
	suite TestSuite,
	run TestRun,
	testName string,
	runContext map[string]any) {
	p.fetchLogsByCorrelationID(runContext)
}

func (p *ElasticSearchPlugin) fetchLogsByCorrelationID(runContext map[string]any) {
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
