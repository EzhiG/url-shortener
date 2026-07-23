package repository

var store = make(map[string]string)

func Save(id, url string) {
	store[id] = url
}

func Get(id string) string {
	return store[id]
}
