package report

type Statistics struct {
	ValidRequests      int
	ValidResponses     int
	InvalidRequests    int
	InvalidResponses   int
	UnparsableMessages int
}

func (s *Statistics) ValidRequestPercentage() float64 {
	requests := s.GetTotalRequest()
	if requests == 0 {
		return 0.0
	}

	return getPercentage(s.ValidRequests, requests)
}

func (s *Statistics) ValidResponsePercentage() float64 {
	responses := s.GetTotalResponses()
	if responses == 0 {
		return 0.0
	}

	return getPercentage(s.ValidResponses, responses)
}

func (s *Statistics) InvalidRequestPercentage() float64 {
	requests := s.GetTotalRequest()
	if requests == 0 {
		return 0.0
	}

	return getPercentage(s.InvalidRequests, requests)
}

func (s *Statistics) InvalidResponsePercentage() float64 {
	responses := s.GetTotalResponses()
	if responses == 0 {
		return 0.0
	}

	return getPercentage(s.InvalidResponses, responses)
}

func (s *Statistics) TotalValidMessagesPercentage() float64 {
	total := s.GetTotal()
	if total == 0 {
		return 0.0
	}

	return getPercentage(s.ValidResponses+s.ValidRequests, total)
}

func (s *Statistics) TotalInvalidMessagesPercentage() float64 {
	total := s.GetTotal()
	if total == 0 {
		return 0.0
	}

	return getPercentage(s.InvalidRequests+s.InvalidResponses, total)
}

func (s *Statistics) GetTotal() int {
	return s.InvalidRequests + s.InvalidResponses + s.ValidRequests + s.ValidResponses
}
func (s *Statistics) GetTotalRequest() int {
	return s.InvalidRequests + s.ValidRequests
}

func (s *Statistics) GetTotalResponses() int {
	return s.InvalidResponses + s.ValidResponses
}

func getPercentage(fraction, total int) float64 {
	return (float64(fraction) / float64(total)) * 100.0
}
