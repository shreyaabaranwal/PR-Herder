package triage

type SizeCategory string

const (
	SizeS  SizeCategory = "S"  
	SizeM  SizeCategory = "M"  
	SizeL  SizeCategory = "L" 
	SizeXL SizeCategory = "XL" 
)

func ClassifySize(additions, deletions int) SizeCategory {
	total := additions + deletions
	switch {
	case total < 10:
		return SizeS
	case total < 100:
		return SizeM
	case total < 500:
		return SizeL
	default:
		return SizeXL
	}
}