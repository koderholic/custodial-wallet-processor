package database

// import (
// 	"fmt"
// 	"log"
// 	"qrCoder/utils"
// 	"strings"
// )

// type migration struct {
// 	Action, Location string
// }

// var (
// 	Migration migration
// 	TableAlterData TableAlter
// 	TableKeys TableAlterKey
// 	TableRefs TableAlterRef
// 	TableDrop map[string][]string
// )

// type tableInfo struct {
// 	Fkey    string `json:"foreignKey"`
// 	Ref     string `json:"reference"`
// 	Rkey    string `json:"referenceKey"`
// 	Actions string `json:"tableActions"`
// }

// // TableAlter ... This defines the table fields to be altered
// type TableAlter struct {
// 	Add  map[string]map[string]string `json:"add"`
// 	Drop map[string]map[string]string `json:"drop"`
// }

// // TableCreate ... This defines the table fields
// type TableAlterKey struct {
// 	Add  map[string]string `json:"add"`
// 	Drop map[string]string `json:"drop"`
// }

// type TableAlterRef struct {
// 	Add  map[string][]tableInfo `json:"add"`
// 	Drop map[string][]tableInfo `json:"drop"`
// }

// func Migrate() {

// 	switch Migration.Action {
// 	case "CREATE":
// 		if err := utils.UnmarshalJsonFile(Migration.Location, &TableStruct); err != nil {
// 			log.Println(err.Error())
// 			return
// 		}
// 		CreateTables()
// 	case "DROP":
// 		if err := utils.UnmarshalJsonFile(Migration.Location, &TableDrop); err != nil {
// 			log.Println(err.Error())
// 			return
// 		}
// 		DropTables()
// 	case "ALTER":
// 		if err := utils.UnmarshalJsonFile(Migration.Location, &TableAlterData); err != nil {
// 			log.Println(err.Error())
// 			return
// 		}
// 		AlterTables()
// 	case "ALTERKEY":
// 		if err := utils.UnmarshalJsonFile(Migration.Location, &TableKeys); err != nil {
// 			log.Println(err.Error())
// 			return
// 		}
// 		AlterPkey()
// 	case "ALTERREF":
// 		if err := utils.UnmarshalJsonFile(Migration.Location, &TableRefs); err != nil {
// 			log.Println(err.Error())
// 			return
// 		}
// 		AlterFkey()
// 	case "TRUNCATE":
// 		if err := utils.UnmarshalJsonFile(Migration.Location, &TableDrop); err != nil {
// 			log.Println(err.Error())
// 			return
// 		}
// 		TruncateTables()
// 	}
// 	return
// }

// func AlterTables() {

// 	done := make(chan bool, 1)
// 	go func(done chan bool) {
// 		for tableName, tableColumns := range TableAlterData.Add {
// 			var execStmt strings.Builder

// 			execStmt.WriteString(fmt.Sprintf("ALTER TABLE IF EXISTS %s ", tableName))

// 			// Build table column fields into the executable statement
// 			for field, fieldType := range tableColumns {
// 				execStmt.WriteString(fmt.Sprintf("ADD COLUMN %s %s,", field, fieldType))
// 			}

// 			finalStmt := strings.TrimSuffix(execStmt.String(), ",") + ";"
// 			//
// 			_, err := DB.DB().Exec(finalStmt)
// 			if err != nil {
// 				log.Println(err.Error())
// 			} else {
// 				log.Println(fmt.Sprintf("Database Table %s Altered successfully", tableName))
// 			}

// 		}
// 		done <- true
// 	}(done)

// 	<-done
// }

// func DropTables() {
// 	done := make(chan bool, 1)
// 	go func(done chan bool) {
// 		print("came")
// 		execStmt := "DROP TABLE IF EXISTS " + strings.Join(TableDrop["drop"], ",") + " RESTRICT;"
// 		_, err := DB.DB().Exec(execStmt)
// 		if err != nil {
// 			fmt.Println("Database Table Drop  error >>> ", err)
// 			log.Println(err.Error())
// 		} else {
// 			log.Println(fmt.Sprintf("Database Table %s Dropped successfully", strings.Join(TableDrop["drop"], ",")))
// 		}
// 		done <- true
// 	}(done)

// 	<-done
// }

// func TruncateTables() {

// 	done := make(chan bool, 1)
// 	go func(done chan bool) {
// 		execStmt := "TRUNCATE TABLE " + strings.Join(TableDrop["truncate"], ",") + " RESTART IDENTITY;"
// 		print(execStmt)
// 		_, err := DB.DB().Exec(execStmt)
// 		if err != nil {
// 			log.Println(err.Error())
// 			return
// 		} else {
// 			log.Println(fmt.Sprintf("Database Table %s Truncated successfully", strings.Join(TableDrop["drop"], ",")))
// 		}
// 		done <- true
// 	}(done)

// 	<-done
// }

// func AlterPkey() {

// 	done := make(chan bool, 1)
// 	go func(done chan bool) {
// 		for tableName, primaryKeys := range TableKeys.Add {
// 			finalStmt := fmt.Sprintf("ALTER TABLE IF EXISTS %s ADD PRIMARY KEY (%s);", strings.ToLower(tableName), primaryKeys)
// 			print(finalStmt)
// 			_, err := DB.DB().Exec(finalStmt)
// 			if err != nil {
// 				log.Println(err.Error())
// 			} else {
// 				log.Println(fmt.Sprintf("Added primary key successfully to table : %s successfully", tableName))
// 			}

// 		}
// 		done <- true
// 	}(done)

// 	<-done
// }

// func AlterFkey() {

// 	done := make(chan bool, 1)
// 	go func(done chan bool) {
// 		for tableName, tableReferences := range TableRefs.Add {
// 			// Build table foreign keys and references (if any) into the executable statement
// 			if len(tableReferences) > 0 {
// 				var execStmt strings.Builder
// 				execStmt.WriteString(fmt.Sprintf("ALTER TABLE IF EXISTS %s ", strings.ToLower(tableName)))

// 				for j := 0; j < len(tableReferences); j++ {
// 					execStmt.WriteString(fmt.Sprintf(`ADD FOREIGN KEY (%s) REFERENCES %s (%s) %s,`, tableReferences[j].Fkey, tableReferences[j].Ref, tableReferences[j].Rkey, tableReferences[j].Actions))
// 				}
// 				finalStmt := strings.TrimSuffix(execStmt.String(), ",")
// 				//
// 				_, err := DB.DB().Exec(finalStmt)
// 				if err != nil {
// 					log.Println(err.Error())
// 					return
// 				} else {
// 					log.Println(fmt.Sprintf("Foreign keys added successfully to table : %s successfully", tableName))
// 				}
// 			}
// 		}
// 		done <- true
// 	}(done)

// 	<-done
// }
