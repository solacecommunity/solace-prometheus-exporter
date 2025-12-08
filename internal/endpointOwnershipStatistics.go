package main

import (
	"crypto/tls"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"log"
)

func main() {
	if len(os.Args) != 4 {
		log.Println("Usage: go run endpointOwnershipStatistics.go  <broker-uri> <username> <password>")
		os.Exit(1)
	}
	var brokerURI = os.Args[1]
	var username = os.Args[2]
	var password = os.Args[3]

	log.SetFlags(0)

	queueStats(brokerURI, username, password)
	topicEndpointStats(brokerURI, username, password)
}

func queueStats(brokerURI string, username string, password string) {
	queueMap := make(map[string][]struct {
		name      string
		vpn       string
		bindCount float64
	})

	type Data struct {
		RPC struct {
			Show struct {
				Queue struct {
					Queues struct {
						Queue []struct {
							QueueName string `xml:"name"`
							Info      struct {
								MsgVpnName string  `xml:"message-vpn"`
								Quota      float64 `xml:"quota"`
								Usage      float64 `xml:"current-spool-usage-in-mb"`
								Owner      string  `xml:"owner"`
								BindCount  float64 `xml:"bind-count"`
							} `xml:"info"`
						} `xml:"queue"`
					} `xml:"queues"`
				} `xml:"queue"`
			} `xml:"show"`
		} `xml:"rpc"`
		MoreCookie struct {
			RPC string `xml:",innerxml"`
		} `xml:"more-cookie"`
		ExecuteResult struct {
			Result string `xml:"code,attr"`
			Reason string `xml:"reason,attr"`
		} `xml:"execute-result"`
	}

	var lastQueueName = ""
	var page = 1
	for nextRequest := "<rpc><show><queue><name>*</name><vpn-name>*</vpn-name><detail/><count/><num-elements>100</num-elements></queue></show></rpc>"; nextRequest != ""; {
		body, err := postHTTP(brokerURI+"/SEMP", "application/xml", nextRequest, username, password, "QueueDetailsSemp1", page)
		page++

		if err != nil {
			log.Println("Can't scrape QueueDetailsSemp1", "err", err, "broker", brokerURI)
			return
		}
		//goland:noinspection ALL
		defer body.Close()
		decoder := xml.NewDecoder(body)
		var target Data
		err = decoder.Decode(&target)
		if err != nil {
			log.Println("Can't decode QueueDetailsSemp1", "err", err, "broker", brokerURI)
			return
		}
		if target.ExecuteResult.Result != "ok" {
			log.Println("Can't scrape QueueDetailsSemp1", "err", err, "broker", brokerURI)
			return
		}

		nextRequest = target.MoreCookie.RPC

		for _, queue := range target.RPC.Show.Queue.Queues.Queue {
			queueKey := queue.Info.MsgVpnName + "___" + queue.QueueName
			if queueKey == lastQueueName {
				continue
			}
			lastQueueName = queueKey

			queueMap[queue.Info.Owner] = append(queueMap[queue.Info.Owner], struct {
				name      string
				vpn       string
				bindCount float64
			}{
				name:      queue.QueueName,
				vpn:       queue.Info.MsgVpnName,
				bindCount: queue.Info.BindCount,
			})
		}

		//goland:noinspection ALL
		body.Close()
	}

	var totalQueues int
	for key, value := range queueMap {
		log.Printf("Owner %s has %d queues\n", key, len(value))
		totalQueues += len(value)
	}
	log.Printf("Total number of queues: %d\n", totalQueues)

	for key, value := range queueMap {
		log.Printf("Owner %s has:\n", key)
		for _, q := range value {
			log.Printf("\t%s\n", q.name)
		}
	}
}

func topicEndpointStats(brokerURI string, username string, password string) {
	queueMap := make(map[string][]struct {
		name      string
		vpn       string
		bindCount float64
	})

	type Data struct {
		RPC struct {
			Show struct {
				TopicEndpoint struct {
					TopicEndpoints struct {
						TopicEndpoint []struct {
							TopicEndpointName string `xml:"name"`
							Info              struct {
								MsgVpnName string  `xml:"message-vpn"`
								Quota      float64 `xml:"quota"`
								Usage      float64 `xml:"current-spool-usage-in-mb"`
								Owner      string  `xml:"owner"`
								BindCount  float64 `xml:"bind-count"`
							} `xml:"info"`
						} `xml:"topic-endpoint"`
					} `xml:"topic-endpoints"`
				} `xml:"topic-endpoint"`
			} `xml:"show"`
		} `xml:"rpc"`
		MoreCookie struct {
			RPC string `xml:",innerxml"`
		} `xml:"more-cookie"`
		ExecuteResult struct {
			Result string `xml:"code,attr"`
			Reason string `xml:"reason,attr"`
		} `xml:"execute-result"`
	}

	var lastQueueName = ""
	var page = 1
	for nextRequest := "<rpc><show><topic-endpoint><name>*</name><vpn-name>*</vpn-name><detail/><count/><num-elements>100</num-elements></topic-endpoint></show></rpc>"; nextRequest != ""; {
		body, err := postHTTP(brokerURI+"/SEMP", "application/xml", nextRequest, username, password, "QueueDetailsSemp1", page)
		page++

		if err != nil {
			log.Println("Can't scrape TopicEndpointDetailsSemp1", "err", err, "broker", brokerURI)
			return
		}
		//goland:noinspection ALL
		defer body.Close()
		decoder := xml.NewDecoder(body)
		var target Data
		err = decoder.Decode(&target)
		if err != nil {
			log.Println("Can't decode TopicEndpointDetailsSemp1", "err", err, "broker", brokerURI)
			return
		}
		if target.ExecuteResult.Result != "ok" {
			log.Println("Can't scrape TopicEndpointDetailsSemp1", "err", err, "broker", brokerURI)
			return
		}

		nextRequest = target.MoreCookie.RPC

		for _, topicEndpoint := range target.RPC.Show.TopicEndpoint.TopicEndpoints.TopicEndpoint {
			queueKey := topicEndpoint.Info.MsgVpnName + "___" + topicEndpoint.TopicEndpointName
			if queueKey == lastQueueName {
				continue
			}
			lastQueueName = queueKey

			queueMap[topicEndpoint.Info.Owner] = append(queueMap[topicEndpoint.Info.Owner], struct {
				name      string
				vpn       string
				bindCount float64
			}{
				name:      topicEndpoint.TopicEndpointName,
				vpn:       topicEndpoint.Info.MsgVpnName,
				bindCount: topicEndpoint.Info.BindCount,
			})
		}

		//goland:noinspection ALL
		body.Close()
	}

	var totalQueues int
	for key, value := range queueMap {
		log.Printf("TopicEndpoint %s has %d queues\n", key, len(value))
		totalQueues += len(value)
	}
	log.Printf("Total number of TopicEndpoints: %d\n", totalQueues)

	for key, value := range queueMap {
		log.Printf("Owner %s has:\n", key)
		for _, q := range value {
			log.Printf("\t%s\n", q.name)
		}
	}
}

func postHTTP(uri string, _ string, body string, username string, password string, logName string, page int) (io.ReadCloser, error) {
	//start := time.Now()
	var httpClient = newHTTPClient()

	req, err := http.NewRequest("POST", uri, strings.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(username, password)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	//var queryDuration = time.Since(start)
	//log.Println("Scraped "+logName, "page", page, "duration", queryDuration)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("HTTP status %d (%s)", resp.StatusCode, http.StatusText(resp.StatusCode))
	}
	return resp.Body, nil
}

func newHTTPClient() http.Client {
	proxy := http.ProxyFromEnvironment

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: false, MinVersion: tls.VersionTLS12},
		Proxy:           proxy,
	}
	client := http.Client{
		Timeout:   30 * time.Second,
		Transport: tr,
	}

	return client
}
