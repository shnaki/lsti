module lsti

go 1.12

require (
	github.com/jessevdk/go-flags v1.4.0
	github.com/jmespath/go-jmespath v0.0.0-20180206201540-c2b33e8439af
	github.com/mattn/go-runewidth v0.0.4 // indirect
	github.com/mattn/go-zglob v0.0.1
	github.com/olekukonko/tablewriter v0.0.1
	github.com/shurcooL/sanitized_anchor_name v1.0.0 // indirect
	gopkg.in/russross/blackfriday.v2 v2.0.1
)

replace gopkg.in/russross/blackfriday.v2 v2.0.1 => github.com/russross/blackfriday/v2 v2.0.1
