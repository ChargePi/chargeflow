package report

import (
	"github.com/stretchr/testify/suite"
	"testing"
)

type statisticsTestSuite struct {
	suite.Suite
}

func (s *statisticsTestSuite) TestValidRequestPercentage() {
	tests := []struct {
		name            string
		validRequests   int
		invalidRequests int
		expected        float64
	}{
		{"100%", 50, 0, 100.0},
		{"50%", 100, 100, 50.0},
		{"0%", 0, 100, 0.0},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			stats := &Statistics{
				ValidRequests:   tt.validRequests,
				InvalidRequests: tt.invalidRequests,
			}

			result := stats.ValidRequestPercentage()
			s.Equal(tt.expected, result)
		})
	}
}

func (s *statisticsTestSuite) TestValidResponsePercentage() {
	tests := []struct {
		name             string
		validResponses   int
		invalidResponses int
		expected         float64
	}{
		{"100%", 50, 0, 100.0},
		{"50%", 100, 100, 50.0},
		{"0%", 0, 200, 0.0},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			stats := &Statistics{
				ValidResponses:   tt.validResponses,
				InvalidResponses: tt.invalidResponses,
			}

			result := stats.ValidResponsePercentage()
			s.Equal(tt.expected, result)
		})
	}
}

func (s *statisticsTestSuite) TestInvalidRequestPercentage() {
	tests := []struct {
		name            string
		invalidRequests int
		validRequests   int
		expected        float64
	}{
		{"100%", 50, 0, 100.0},
		{"50%", 100, 100, 50.0},
		{"0%", 0, 200, 0.0},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			stats := &Statistics{
				InvalidRequests: tt.invalidRequests,
				ValidRequests:   tt.validRequests,
			}

			result := stats.InvalidRequestPercentage()
			s.Equal(tt.expected, result)
		})
	}
}

func (s *statisticsTestSuite) TestTotalValidMessagesPercentage() {
	tests := []struct {
		name             string
		invalidResponses int
		invalidRequests  int
		validResponses   int
		validRequests    int
		expected         float64
	}{
		{"100%", 0, 0, 100, 100, 100.0},
		{"100%", 0, 0, 100, 10, 100.0},
		{"50%", 100, 100, 100, 100, 50.0},
		{"0%", 100, 100, 0, 0, 0.0},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			stats := &Statistics{
				InvalidResponses: tt.invalidResponses,
				InvalidRequests:  tt.invalidRequests,
				ValidResponses:   tt.validResponses,
				ValidRequests:    tt.validRequests,
			}

			result := stats.TotalValidMessagesPercentage()
			s.Equal(tt.expected, result)
		})
	}
}

func (s *statisticsTestSuite) TestTotalInvalidMessagesPercentage() {
	tests := []struct {
		name                  string
		invalidResponsesTotal int
		invalidRequestsTotal  int
		validResponsesTotal   int
		validRequestsTotal    int
		expected              float64
	}{
		{"100%", 50, 50, 0, 0, 100.0},
		{"50%", 100, 100, 100, 100, 50.0},
		{"0%", 0, 0, 200, 200, 0.0},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			stats := &Statistics{
				InvalidResponses: tt.invalidResponsesTotal,
				InvalidRequests:  tt.invalidRequestsTotal,
				ValidResponses:   tt.validResponsesTotal,
				ValidRequests:    tt.validRequestsTotal,
			}

			result := stats.TotalInvalidMessagesPercentage()
			s.Equal(tt.expected, result)
		})
	}
}

func (s *statisticsTestSuite) TestGetTotal() {
	tests := []struct {
		name                  string
		invalidResponsesTotal int
		invalidRequestsTotal  int
		validResponsesTotal   int
		validRequestsTotal    int
		expectedTotal         int
	}{
		{"Zero totals", 0, 0, 0, 0, 0},
		{"All valid", 0, 0, 100, 100, 200},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			stats := &Statistics{
				InvalidResponses: tt.invalidResponsesTotal,
				InvalidRequests:  tt.invalidRequestsTotal,
				ValidResponses:   tt.validResponsesTotal,
				ValidRequests:    tt.validRequestsTotal,
			}

			result := stats.GetTotal()
			s.Equal(tt.expectedTotal, result)
		})
	}
}

func (s *statisticsTestSuite) Test_getPercentage() {
	fraction := 50
	total := 200
	percentage := getPercentage(fraction, total)
	s.Equal(25.0, percentage)
}

func TestStatistics(t *testing.T) {
	suite.Run(t, new(statisticsTestSuite))
}
