# Lets Go 🚀

[![wakatime](https://wakatime.com/badge/github/edmealem-k/learn-golang.svg)](https://wakatime.com/badge/github/edmealem-k/learn-golang)

A hands-on repository documenting my journey of learning the Go programming language from various courses, blogs, and real-world projects. This repository contains code snippets, syntax experiments, notes, and mini-applications built along the way.

---

## 📂 Repository Structure

| Directory | Description | Status |
| :--- | :--- | :--- |
| [`go-blog/`](./go-blog) | Notes and code following Karan Pratap Singh's comprehensive Go course. | 🔄 In Progress |
| [`rest-api/`](./rest-api) | RESTful Event Management API built with Gin and SQLite. | ✅ Completed |
| [`price-calculator/`](./price-calculator) | Tax and price calculation tool demonstrating interfaces & file I/O. | ✅ Completed |
| [`basics/`](./basics) | Fundamental Go syntax, flow controls, data structures, and CLI calculator. | ✅ Completed |
| [`hello/`](./hello) | Initial setup and workspace verification. | ✅ Completed |

---

## 📖 Modules & Projects Overview

### 1. 📘 `go-blog/` — Learn Go: The Complete Course
Hands-on implementation and notes based on [Learn Go: The Complete Course](https://www.karanpratapsingh.com/blog/learn-go-the-complete-course) by Karan Pratap Singh *(currently studying & updating)*.

**Topics covered:**
- **Variables, Constants & Data Types:** Zero values, implicit inference, type conversion.
- **Flow Control:** If/Else, switch statements, `for` loops (standard, range, while-style, infinite).
- **Functions:** Named returns, multiple return values, first-class functions, closures, variadic parameters.
- **Special Keywords & Lifecycle:** `init()`, `defer` stacks (LIFO cleanup).
- **Packages & Modules:** Exported vs unexported identifiers, `go mod`, external dependencies (`zerolog`).
- **Memory & Pointers:** Memory addresses, dereferencing, pointer-to-pointer, pass-by-reference.
- **Structs & Methods:** Struct literals, composition/embedding, struct tags (`json:`), value vs pointer receivers.
- **Collections:** Arrays vs Slices, slice headers (pointer, length, capacity), `copy()`, `append()`, Maps and iteration.
- **Interfaces & Polymorphism:** Implicit implementation, empty interface (`any`), type assertions, type switches, interface embedding.

---

### 2. 🌐 `rest-api/` — Events REST API
A full-featured backend API for managing events and user registrations.

- **Framework:** [Gin Web Framework](https://github.com/gin-gonic/gin)
- **Database:** SQLite with raw SQL queries (`database/sql` & `go-sqlite3`)
- **Key Features:**
  - Event CRUD operations (`GET /events`, `GET /events/:id`, `POST /events`, `PUT /events/:id`, `DELETE /events/:id`)
  - User signup and authentication utilities (`bcrypt` password hashing)
  - Modular project structure: `routes/`, `models/`, `db/`, and `utils/`

---

### 3. 🏷️ `price-calculator/` — Price & Tax Calculation Package
A project focused on Go interfaces, decoupled architecture, and file processing.

- **Key Highlights:**
  - **Interface-Driven Design:** `IOManager` interface decoupling file handling (`filemanager`) and CLI handling (`cmdmanager`).
  - Reads prices from `prices.txt`, applies multiple tax rates dynamically, and writes formatted output JSON files (`result_*.json`).
  - Error handling and string/float conversions.

---

### 4. 🧮 `basics/` — Core Fundamentals
Practical exercises covering Go syntax essentials and early experiments:
- Variables, conditionals, pointers, arrays, slices, and structs.
- Interactive CLI Calculator with operations (+, -, *, /) and input parsing (`fmt.Scanf`).

---

## 🛠️ Getting Started

### Prerequisites
- [Go](https://go.dev/dl/) (version 1.22+ recommended)

### Running any module

Each folder contains its own module or Go files. Navigate to the desired directory and run:

```bash
# Example: Run go-blog notes
cd go-blog
go run main.go

# Example: Run REST API
cd rest-api
go run main.go

# Example: Run Price Calculator
cd price-calculator
go run main.go

# Example: Run Basics
cd basics
go run main.go
```

---

## 📚 Learning Resources

- 🌐 [Learn Go: The Complete Course - Karan Pratap Singh](https://www.karanpratapsingh.com/blog/learn-go-the-complete-course)
- 📖 [A Tour of Go](https://go.dev/tour/)
- 📘 [Effective Go](https://go.dev/doc/effective_go)
- 📑 [Go Standard Library Documentation](https://pkg.go.dev/std)
