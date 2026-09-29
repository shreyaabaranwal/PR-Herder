package triage

type CheckRunObservation struct {
	CommitSHA  string
	Conclusion string 
}


type FlakyVerdict struct {
	IsFlaky bool
	
	FailThenPassRate float64
	SampleSize       int
}


const minSampleSize = 3


const flakyThreshold = 0.5


func ClassifyFlaky(observations []CheckRunObservation) FlakyVerdict {

	type commitRun struct {
		results []string
	}
	byCommit := make(map[string]*commitRun)
	var commitOrder []string

	for _, obs := range observations {
		run, exists := byCommit[obs.CommitSHA]
		if !exists {
			run = &commitRun{}
			byCommit[obs.CommitSHA] = run
			commitOrder = append(commitOrder, obs.CommitSHA)
		}
		run.results = append(run.results, obs.Conclusion)
	}

	commitsWithFailure := 0
	commitsFailThenPass := 0

	for _, sha := range commitOrder {
		run := byCommit[sha]
		hadFailure := false
		hadFailureThenSuccess := false
		sawFailure := false

		for _, result := range run.results {
			if result == "failure" {
				hadFailure = true
				sawFailure = true
			} else if result == "success" && sawFailure {
				hadFailureThenSuccess = true
			}
		}

		if hadFailure {
			commitsWithFailure++
		}
		if hadFailureThenSuccess {
			commitsFailThenPass++
		}
	}

	sampleSize := len(commitOrder)

	if commitsWithFailure == 0 {
		
		return FlakyVerdict{IsFlaky: false, FailThenPassRate: 0, SampleSize: sampleSize}
	}

	rate := float64(commitsFailThenPass) / float64(commitsWithFailure)

	isFlaky := sampleSize >= minSampleSize && rate >= flakyThreshold

	return FlakyVerdict{
		IsFlaky:          isFlaky,
		FailThenPassRate: rate,
		SampleSize:       sampleSize,
	}
}
