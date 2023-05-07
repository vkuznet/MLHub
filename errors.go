package main

import "fmt"

const (
	GenericError        = iota + 100 // generic DBS error
	DatabaseError                    // 101 database error
	BadRequest                       // 102 bad request
	JsonMarshal                      // 103 json.Marshal error
	MetaDataRecordError              // 104 Meta data record error
	MetaDataError                    // 105 generic Meta data error
	FileIOError                      // 106 file IO error
	InsertError                      // 107 insert error
)

// helper function to return human error message for given MLHub error code
func errorMessage(code int) string {
	if code == 0 {
		return ""
	} else if code == 101 {
		return "database error"
	} else if code == 102 {
		return "bad request"
	} else if code == 103 {
		return "JSON marshal error"
	} else if code == 104 {
		return "MetaData record error"
	} else if code == 105 {
		return "MetaData error"
	} else if code == 106 {
		return "file IO error"
	} else if code == 107 {
		return "Insert error"
	} else {
		return fmt.Sprintf("Not Implemented error for code %d", code)
	}
}
