package mars

func makeSim94() *MARS {
	return NewMARS(8000, 8000, 80000, 8000, 8000, false)
}

func makeSim88() *MARS {
	return NewMARS(8000, 8000, 80000, 8000, 8000, true)
}
