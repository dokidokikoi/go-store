package es

// 对应 elaticsearch 搜索结果结构
type hit struct {
	Source Metadata `json:"_source"`
}

type searchResult struct {
	Hits struct {
		Total int
		Hits  []hit
	}
}
