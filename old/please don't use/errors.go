package pure

type errorKind int

const (
	none errorKind = iota
	strictModeTab
	incorrectTab
	keyAlreadyDefined
	groupAlreadyDefined
	defaualtValueDoesNotExist

	arrayMultipleTypes
	schemaDefinitionsInConfigFile

	nonexsistantType
	configDefinitionsInSchema

	valueIncorrectType
	unexpectedKey
	keyNotFound
	arrayIncorrectType

	keyNameTooLarge
	stringValueToLarge
	numberTooLarge
	roundedDecimal
	arrayTooLarge
	tooManyKeys
	nestedTooDeep
	tooManyImportedFiles
)

type PureError struct {
	message string
	start   int
	end     int
	kind    int
}
