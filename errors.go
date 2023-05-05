package main

const (
	GenericError        = iota + 100 // generic DBS error
	DatabaseError                    // 101 database error
	BadRequest                       // 102 bad request
	JsonMarshal                      // 103 json.Marshal error
	MetaDataRecordError              // 104 Meta data record error
	MetaDataError                    // 105 generic Meta data error
	FileIOError                      // 106 file IO error
)
