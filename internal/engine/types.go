package engine

type BakeProgress struct {
	DownloadedBytes float64
	TotalBytes      *float64
	Speed           float64
	ETA             float64
	Part            int
	TotalParts      int
	PlaylistItem    int
	PlaylistTotal   int
	Connections     int
	Seeders         *int
}

type BakeHandlers struct {
	OnProgress   func(progress BakeProgress)
	OnProcessing func()
}
