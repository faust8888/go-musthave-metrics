// Command staticlint is a project multichecker.
//
// # Запуск
//
// Из корня репозитория:
//
//	go run ./cmd/staticlint ./...
//
// Или сначала собрать бинарник:
//
//	go build -o staticlint ./cmd/staticlint
//	./staticlint ./...
//
// Аргументы те же, что у go vet: пути пакетов, флаги анализаторов
// (`-NAME` включает/выключает конкретный анализатор, `-NAME.debug` и т.д.).
// Примеры:
//
//	./staticlint ./internal/handler
//	./staticlint -exitcheck=false ./cmd/...
//
// Исходный код проекта должен проходить проверку без диагностик.
//
// # Состав
//
//  1. Стандартные анализаторы golang.org/x/tools/go/analysis/passes
//     (набор go vet и близкие к нему проверки):
//
//     - appends — append с одним и тем же слайсом в нескольких аргументах;
//     - asmdecl — соответствие сборок и Go-деклараций;
//     - assign — бесполезные присваивания;
//     - atomic — некорректное использование sync/atomic;
//     - bools — лишние булевы выражения;
//     - buildtag — синтаксис build tags;
//     - cgocall — нарушения правил cgo;
//     - composite — незаданённые ключи в составных литералах неэкспортируемых структур;
//     - copylock — копирование значений с locks;
//     - deepequalerrors — сравнение ошибок через reflect.DeepEqual;
//     - defers — подозрительные defer (например, defer f.Close без скобок);
//     - directive — директивы //go:;
//     - errorsas — второй аргумент errors.As должен быть указателем на error-реализацию;
//     - httpresponse — использование HTTP-ответа до проверки ошибки;
//     - ifaceassert — невозможные приведения интерфейсов;
//     - loopclosure — захват переменной цикла в горутине/функции;
//     - lostcancel — потерянный context.CancelFunc;
//     - nilfunc — сравнение функции с nil;
//     - printf — форматные строки Print-подобных функций;
//     - shift — сдвиги, выходящие за ширину типа;
//     - sigchanyzer — небуферизованный chan os.Signal;
//     - slog — некорректные ключи log/slog;
//     - sortslice — sort.Slice с не-слайсом;
//     - stdmethods — сигнатуры стандартных методов (Error, MarshalJSON, …);
//     - stdversion — символы стандартной библиотеки новее языка модуля;
//     - stringintconv — подозрительные преобразования int ↔ string;
//     - structtag — синтаксис struct tags;
//     - testinggoroutine — вызов t.Fatal из посторонней горутины;
//     - tests — опечатки в именах Test/Benchmark/Fuzz/Example;
//     - timeformat — time.Format с ошибочным эталоном 2006-02-01;
//     - unmarshal — Unmarshal в не-указатель;
//     - unreachable — недостижимый код;
//     - unsafeptr — некорректное преобразование в unsafe.Pointer;
//     - unusedresult — игнорирование результата функций вроде fmt.Errorf, context.WithCancel.
//
//  2. Все анализаторы класса SA пакета honnef.co/go/tools/staticcheck
//     (проверки на реальные ошибки: неверное использование API, бесполезные
//     условия, гонки вокруг locks, проблемы с HTTP, контекстом, регулярками и т.д.).
//     Имена имеют вид SA1xxx–SA9xxx; полный список — https://staticcheck.io/docs/checks/#SA.
//
//  3. Анализатор класса S (simple) — S1012: заменять time.Now().Sub(x) на time.Since(x).
//
//  4. Публичные анализаторы:
//
//     - ineffassign (github.com/gordonklaus/ineffassign) — присваивания, результат
//     которых никогда не используется;
//     - bodyclose (github.com/timakin/bodyclose) — http.Response.Body должен быть закрыт.
//
//  5. Собственный анализатор exitcheck: прямой вызов os.Exit в функции main
//     пакета main запрещён (в том числе через алиас пакета os и во вложенных
//     литералах внутри main). os.Exit в других функциях пакета main допустим —
//     вынесите завершение процесса из main, если оно нужно.
package main

import "golang.org/x/tools/go/analysis/multichecker"

func main() {
	multichecker.Main(analyzers()...)
}
