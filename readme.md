# Постановка задачи

В этом задании необходимо написать кодогенератор, который ищет методы структуры, помеченные спец меткой и генерирует для них следующий код:
* http-обёртки для этих методов
* проверку авторизации
* проверки метода (GET/POST)
* валидацию параметров
* заполнение структуры с параметрами метода
* обработку неизвестных ошибок
 
Запуск выглядит так: `go build handlers_gen/* && ./codegen api.go api_handlers.go`. 
 
Все данные - имена полей, доступные значения, граничные значения - берутся из самой струкруты, `struct tags apivalidator` и кода, который мы парсим.
 
Кодогенератор работает универсально для любых полей и значений из тех что ему известны. Кодогенератор может отработать на неизвестном вам коде, аналогичном `api.go`.
 
Cчитаем `type ApiError struct` при проверке ошибки, что это какая-то общеизвестная структура.
 
Кодогенератор умеет обрабатывать следующие типы полей структуры:
* `int`
* `string`
 
Нам доступны следующие метки валидатора-заполнятора `apivalidator`:
* `required` - поле не должно быть пустым (не должно иметь значение по-умолчанию)
* `paramname` - если указано - то брать из параметра с этим именем, иначе `lowercase` от имени
* `enum` - "одно из"
* `default` - если указано и приходит пустое значение (значение по-умолчанию) - устанавливать то что написано указано в `default`
* `min` - >= X для типа `int`, для строк `len(str)` >=
* `max` - <= X для типа `int`
 
Порядок следования ошибок в тестах:
* наличие метода (в `ServeHTTP`)
* метод (POST)
* авторизация
* параметры в порядке следования в структуре
 
Авторизация проверяется просто на то что в хедере пришло значение `100500`
 
Сгенерённый имеет примерно такую цепочку
 
`ServeHTTP` - принимает все методы из мультиплексора, если нашлось - вызывает `handler$methodName`, если нет - говорит `404`
`handler$methodName` - обёртка над методом структуры `$methodName` - осуществляет все проверки, выводит ошибки или результат в формате `JSON`
`$methodName` - непосредственно метод структуры для которого мы генерируем код и, который парсим. Имеет префикс `apigen:api` за которым следует `json` с именем метода, типом и требованием авторизации. Его генерировать не нужно, он уже есть.
 
``` go
type SomeStructName struct{}
 
func (h *SomeStructName ) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    switch r.URL.Path {
    case "...":
        h.wrapperDoSomeJob(w, r)
    default:
        // 404
    }
}
 
func (h *SomeStructName ) wrapperDoSomeJob() {
    // заполнение структуры params
    // валидирование параметров
    res, err := h.DoSomeJob(ctx, params)
    // прочие обработки
}
```
 
По структуре кодогенератора - надо найти все методы, для каждого метода сгенерировать валидацию входящих параметров и прочие проверки в `handler$methodName`, для пачки методов структуры сгенерировать обвязку в `ServeHTTP`
  
Что надо парсить в ast:
* `node.Decls` -> `ast.FuncDecl` - это методы. У проверяем что есть метка и начать генерировать для него обёртку
* `node.Decls` -> `ast.GenDecl` -> `spec.(*ast.TypeSpec)` + `currType.Type.(*ast.StructType)` - это структура. Она нужна чтобы по ней генерить валидацию для метода, который мы нашли в проедыдущем пункте
* https://golang.org/pkg/go/ast/#FuncDecl - тут смотрите к какой структуре относится метод

go build handlers_gen/* && ./codegen.exe api.go api_handlers.go
# запуск тестов
go test -v
```
