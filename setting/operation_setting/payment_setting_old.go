/**
此文件为旧版支付设置文件，如需增加新的参数、变量等，请在 payment_setting.go 中添加
This file is the old version of the payment settings file. If you need to add new parameters, variables, etc., please add them in payment_setting.go
*/

package operation_setting

// CustomCallbackAddress lets an operator override the public callback base
// address used by payment integrations. It is not tied to any single gateway.
var CustomCallbackAddress = ""

var Price = 1000.0
var MinTopUp = 1
var USDExchangeRate = 7.3
