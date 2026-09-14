# A Reproducibility and Validity Study of God Class Detection in Go

This project contains modifications to GodExpo as part of analysis
and investigation of of GodExpo's correctness.

---

# GodExpo

GodExpo is a God Struct smell detector for Golang.

## Getting Started

These instructions will get you a copy of the project up and running on your local machine for development and testing purposes.

### Prerequisites

Golang must be installed (>= v1.10). To install golang in ubuntu run-

```Bash
sudo apt-get install golang
```

### Compiling

At first, download the project-

```Bash
git clone https://github.com/Grant-Nelson/GodExpo
```

Build the project with

```Bash
go build -o godExpo  main.go
```

## Running the Tool

You can now run the tool on go projects to-

* Show struct summary
* Find God Structs
* Show evolution of god structs
* Set custom thresholds for metric calculations

### 1. Show Struct Summary

```Bash
./godExpo -f path_to_file.go
```

Sample output-

![Struct sumary](img/file.png)

### 2. Find God Structs

```Bash
./godExpo -d path_to_directory/
```

Sample output-

![Find god structs](img/project.png)

### 3. Show evolution of god structs

```Bash
./godExpo -e path_to_directory/
```
* Directory should have different versions of a project

Sample output-

![God evolution](img/evolution_small.png)


### 4. Set Custom Thresholds for Metric Calculations

Set custom WMC

```Bash
./godExpo -wmc 50 -d path_to_directory/
```

Set custom ATFD

```Bash
./godExpo -atfd 10 -d path_to_directory/
```

Set custom TCC

```Bash
./godExpo -tcc 0.5 -d path_to_directory/
```

## Authors

* [**Rafed Muhammad Yasir**](https://github.com/rafed123)
* [**Moumita Asad**](https://github.com/mou23)


## License

This project is licensed under the MIT License - see the [LICENSE.md](LICENSE.md) file for details.

