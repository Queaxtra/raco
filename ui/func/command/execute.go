package command

import (
	"raco/http"
	"raco/model"

	tea "github.com/charmbracelet/bubbletea"
)

type RequestExecutedMsg struct {
	Response         *model.Response
	Error            string
	AssertionResults []model.AssertionResult
}

func Execute(client *http.Client, req *model.Request, env *model.Environment) tea.Cmd {
	return func() tea.Msg {
		if req == nil {
			return RequestExecutedMsg{Response: nil}
		}

		processedReq := *req
		processedReq.URL = http.ReplaceEnvVars(req.URL, env)
		processedReq.Assertions = req.Assertions
		processedReq.Extractors = req.Extractors

		resp, err := client.Execute(&processedReq)
		if err != nil {
			return RequestExecutedMsg{Response: nil, Error: err.Error()}
		}

		results := make([]model.AssertionResult, 0)
		if len(req.Assertions) > 0 {
			for _, assertion := range req.Assertions {
				result := model.ValidateAssertion(assertion, resp)
				results = append(results, result)
			}
		}

		if len(req.Extractors) > 0 {
			if env != nil {
				for _, extractor := range req.Extractors {
					_ = model.ExtractValue(extractor, resp, env)
				}
			}
		}

		return RequestExecutedMsg{Response: resp, AssertionResults: results}
	}
}
