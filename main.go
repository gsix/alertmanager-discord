package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"gopkg.in/yaml.v2"
)

type alertManOut struct {
	Alerts []struct {
		Annotations struct {
			Description string `json:"description"`
			Summary     string `json:"summary"`
		} `json:"annotations"`
		EndsAt       string            `json:"endsAt"`
		GeneratorURL string            `json:"generatorURL"`
		Labels       map[string]string `json:"labels"`
		StartsAt     string            `json:"startsAt"`
		Status       string            `json:"status"`
	} `json:"alerts"`
	CommonAnnotations struct {
		Summary string `json:"summary"`
	} `json:"commonAnnotations"`
	CommonLabels struct {
		Alertname string `json:"alertname"`
	} `json:"commonLabels"`
	ExternalURL string `json:"externalURL"`
	GroupKey    string `json:"groupKey"`
	GroupLabels struct {
		Alertname string `json:"alertname"`
	} `json:"groupLabels"`
	Receiver string `json:"receiver"`
	Status   string `json:"status"`
	Version  string `json:"version"`
}

type discordOut struct {
	Content string `json:"content"`
	Name    string `json:"username"`
}

type wHooks struct {
	Hooks []struct {
		Project string `yaml:"project"`
		Env     []struct {
			Name string `yaml:"name"`
			Hook string `yaml:"hook"`
		} `yaml:"env"`
	} `yaml:"hooks"`
}

func findWHook(projectName, projectEnv string) string {
	var configs wHooks
	data, err := ioutil.ReadFile("./webhooks.yml")

	if err != nil {
		fmt.Println(err)
	}

	err = yaml.Unmarshal(data, &configs)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	for _, config := range configs.Hooks {
		if config.Project == projectName {
			for _, env := range config.Env {
				if env.Name == projectEnv {
					return env.Hook
				}
			}
		}
	}
	return ""
}

func main() {
	fmt.Fprintf(os.Stdout, "info: Listening on 0.0.0.0:9094\n")
	http.ListenAndServe(":9094", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			panic(err)
		}

		amo := alertManOut{}
		err = json.Unmarshal(b, &amo)
		if err != nil {
			panic(err)
		}

		emoji := "bell"
		// Let's have a nice emoji
		if strings.ToUpper(amo.Status) == "FIRING" {
			emoji = ":airplane_arriving: :fire:"
		} else {
			emoji = ":wine_glass: :cheese:"
		}

		for _, alert := range amo.Alerts {
			DO := discordOut{
				Name: amo.Status,
			}
			Content := "```"
			if alert.Annotations.Summary != "" {
				Content = fmt.Sprintf("%s %s\n```\n", emoji, alert.Annotations.Summary)
			}

			for _, alert := range amo.Alerts {
				realname := alert.Labels["instance"]
				if strings.Contains(realname, "localhost") && alert.Labels["exported_instance"] != "" {
					realname = alert.Labels["exported_instance"]
				}
				Content += fmt.Sprintf("[%s]: %s on %s\n%s\n\n", strings.ToUpper(amo.Status), alert.Labels["alertname"], realname, alert.Annotations.Description)
			}
			DO.Content = Content + "```"

			whURL := findWHook(alert.Labels["project"], alert.Labels["env"])
			DOD, _ := json.Marshal(DO)
			http.Post(whURL, "application/json", bytes.NewReader(DOD))
		}
	}))
}
