module example.com/feedparcer

go 1.17

require (
	example.org/uploader v0.0.0
	example.org/parser v0.0.0

)

replace (
	example.org/uploader => ./uploader
	example.org/parser => ./parser
)

