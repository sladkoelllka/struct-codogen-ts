package parser

// Field описывает поле структуры
type Field struct {
	Name     string // Имя поля в Go структуре
	Type     string // Тип поля (в строковом виде, например: string, *User, []int)
	JSONTag  string // Значение json тега (например: "user_id,omitempty")
	UriTag   string // Значение uri тега (например: "id")
	FormTag  string // Значение form тега (например: "name")
	Binding  string // Значение binding тега gin (например: "required,email")
	Optional bool   // Является ли поле опциональным (используется при генерации типов)
	Embedded bool   // Встроенное поле (анонимный embedding)
}

// Struct представляет определение Go структуры
type Struct struct {
	Name      string  // Имя структуры или alias-типа
	Package   string  // Имя пакета, откуда была прочитана структура
	Fields    []Field // Список полей структуры
	Comment   string  // Комментарий перед struct (если есть)
	AliasType string  // Целевой тип для alias (например, dto.LoginResponse)
}

// File представляет распарсенный Go файл
type File struct {
	Path     string            // Полный путь к исходному Go файлу
	Package  string            // Имя пакета файла
	Dir      string            // Директория файла (основывается на пути к файлу)
	Entity   string            // Entity path (например, user, report/whitebox)
	Type     string            // Тип файла (request, response и т.д., извлекается из имени файла)
	Imports  map[string]string // Импорты файла: локальное имя -> полный Go import path
	Structs  []Struct          // Все найденные структуры в файле
	Generate bool              // Нужно ли генерировать отдельный TS файл для этого Go файла
}
