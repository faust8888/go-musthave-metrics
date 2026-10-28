# go-musthave-metrics-tpl

Шаблон репозитория для трека «Сервер сбора метрик и алертинга».

## Начало работы

1. Склонируйте репозиторий в любую подходящую директорию на вашем компьютере.
2. В корне репозитория выполните команду `go mod init <name>` (где `<name>` — адрес вашего репозитория на GitHub без префикса `https://`) для создания модуля.

## Обновление шаблона

Чтобы иметь возможность получать обновления автотестов и других частей шаблона, выполните команду:

```
git remote add -m v2 template https://github.com/Yandex-Practicum/go-musthave-metrics-tpl.git
```

Для обновления кода автотестов выполните команду:

```
git fetch template && git checkout template/v2 .github
```

Затем добавьте полученные изменения в свой репозиторий.

## Запуск автотестов

Для успешного запуска автотестов называйте ветки `iter<number>`, где `<number>` — порядковый номер инкремента. Например, в ветке с названием `iter4` запустятся автотесты для инкрементов с первого по четвёртый.

При мёрже ветки с инкрементом в основную ветку `main` будут запускаться все автотесты.

Подробнее про локальный и автоматический запуск читайте в [README автотестов](https://github.com/Yandex-Practicum/go-autotests).

## Структура проекта

Приведённая в этом репозитории структура проекта является рекомендуемой, но не обязательной.

Это лишь пример организации кода, который поможет вам в реализации сервиса.

При необходимости можно вносить изменения в структуру проекта, использовать любые библиотеки и предпочитаемые структурные паттерны организации кода приложения, например:
- **DDD** (Domain-Driven Design)
- **Clean Architecture**
- **Hexagonal Architecture**
- **Layered Architecture**

## Бенчмарки и профилирование памяти

Бенчмарки важнейших компонентов:

```
go test -bench=. -benchmem ./internal/handler ./internal/agent ./internal/hash
```

Профили аллокаций сняты командой `go test -bench=. -memprofile=profiles/<name>.pprof ./internal/handler`:
- `profiles/base.pprof` — до оптимизаций;
- `profiles/result.pprof` — после пула gzip-писателей, `json.Unmarshal` вместо `Decoder` и прямой записи HTML без `fmt.Fprintf`.

Сравнение профилей:

```
go tool pprof -top -diff_base=profiles/base.pprof profiles/result.pprof
```

```
File: handler.test
Type: alloc_space
Time: 2026-10-15 00:04:06 MSK
Showing nodes accounting for -3332.11MB, 53.18% of 6265.20MB total
Dropped 91 nodes (cum <= 31.33MB)
      flat  flat%   sum%        cum   cum%
-1248.99MB 19.94% 19.94% -2261.27MB 36.09%  compress/flate.NewWriter (inline)
 -635.69MB 10.15% 30.08% -1012.28MB 16.16%  compress/flate.(*compressor).init
 -406.52MB  6.49% 36.57%  -406.52MB  6.49%  bufio.NewReaderSize (inline)
 -369.58MB  5.90% 42.47%  -369.58MB  5.90%  compress/flate.newDeflateFast (inline)
 -272.28MB  4.35% 46.82%  -272.28MB  4.35%  strings.(*Builder).Write
 -249.20MB  3.98% 50.79%  -249.20MB  3.98%  encoding/json.(*Decoder).refill
  166.09MB  2.65% 48.14%   166.09MB  2.65%  io.ReadAll
 -118.51MB  1.89% 50.03%  -419.22MB  6.69%  github.com/faust8888/go-musthave-metrics/internal/handler_test.BenchmarkIndex.Index.func1
 -107.03MB  1.71% 51.74%  -107.03MB  1.71%  encoding/json.NewDecoder (partial-inline)
```

Отрицательные значения означают, что после оптимизаций объём аллокаций снизился. Основной выигрыш дал пул `gzip.Writer`: в `BenchmarkGzipIndex` расход памяти упал с ~1.2 MB/op до ~18 KB/op.
