package main

import (
	"qq-zone/app/controllers"
)

func main() {
	(&controllers.QzoneController{}).Cmd()
}