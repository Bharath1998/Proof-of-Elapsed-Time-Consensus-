 // package main
package myalgo
// import (
// 	"fmt"
//     "os"
// 	"strconv"
// 	"encoding/json"
// 	"io/ioutil"
// )
// type Problem struct {
// 	Index    int    `json:"index"`
// 	Equation string `json:"equation"`
// }

// func getProblemFromHeader () (Problem, int64){
// 	runes := []rune("0x023d")
// 	index_in_hash := string(runes[0:3])
// 	index_in_decimal, _ := strconv.ParseInt(index_in_hash , 0, 64)
// 	index_in_decimal = index_in_decimal % 10
// 	fmt.Println(index_in_decimal,problems)
// 	return problems[index_in_decimal], index_in_decimal
// }
// var problems []Problem=getProblems()
// func getProblems() []Problem {

// 	raw, err := ioutil.ReadFile("./problems.json")
// 	if err != nil {
// 		fmt.Println(err.Error())
// 		os.Exit(1)
// 	}

// 	var c []Problem
// 	json.Unmarshal(raw, &c)
// 	return c
// }
// func main(){
// 	fmt.Println(problems[9])
// 	fmt.Println(getProblemFromHeader())
// }