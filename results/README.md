# Results

These results were gathered using the following commands

## Hugo v0.50

```Bash
go run main.go -d ../../gohugoio/hugo > ./results/hugoFull.txt 2>&1
go run main.go -bc -st -sv -d ../../gohugoio/hugo > ./results/hugoTrim.txt 2>&1
go run main.go -bc -mp -st -sv -d ../../gohugoio/hugo > ./results/hugoPkgPath.txt 2>&1
```

where `../..gohugoio/hugo` is commit `f5be59920461366d4481cf7536b2d2bd42a2c75e`, tag `v0.50`

## Mattermost v5.6.0

```Bash
go run main.go -d ../../mattermost/mattermost > ./results/mattermostFull.txt 2>&1
go run main.go -bc -st -sv -d ../../mattermost/mattermost > ./results/mattermostTrim.txt 2>&1
go run main.go -bc -mp -st -sv -d ../../mattermost/mattermost > ./results/mattermostPkgPath.txt 2>&1
```

where `../../mattermost/mattermost` is commit `0c1207215852a8726c3f09dea157d597fec368df`, tag `v5.6.0`
