module Api.Point exposing
    ( Point
    , blankMajicValue
    , clearText
    , decode
    , empty
    , encode
    , encodeList
    , filterSpecialPoints
    , get
    , getLatest
    , getText
    , getValue
    , newText
    , newValue
    , renderPoint
    , typeAddress
    , typeAppVersion
    , typeBaud
    , typeClientServer
    , typeCmdPending
    , typeDataFormat
    , typeDebug
    , typeDescription
    , typeEmail
    , typeFirstName
    , typeHwVersion
    , typeID
    , typeLastName
    , typeModbusIOType
    , typeOSVersion
    , typeOffset
    , typePass
    , typePhone
    , typePort
    , typeScale
    , typeStartApp
    , typeStartSystem
    , typeSwUpdateError
    , typeSwUpdatePercComplete
    , typeSwUpdateRunning
    , typeSwUpdateState
    , typeSysState
    , typeUnits
    , typeUpdateApp
    , typeUpdateOS
    , typeValue
    , typeValueSet
    , updatePoint
    , updatePoints
    , valueClient
    , valueFLOAT32
    , valueINT16
    , valueINT32
    , valueModbusCoil
    , valueModbusDiscreteInput
    , valueModbusHoldingRegister
    , valueModbusInputRegister
    , valueServer
    , valueUINT16
    , valueUINT32
    )

import Iso8601
import Json.Decode as Decode
import Json.Decode.Extra
import Json.Decode.Pipeline exposing (optional)
import Json.Encode
import List.Extra
import Round
import Time


typeDescription : String
typeDescription =
    "description"


typeScale : String
typeScale =
    "scale"


typeOffset : String
typeOffset =
    "offset"


typeUnits : String
typeUnits =
    "units"


typeValue : String
typeValue =
    "value"


typeValueSet : String
typeValueSet =
    "valueSet"


typeCmdPending : String
typeCmdPending =
    "cmdPending"


typeSwUpdateState : String
typeSwUpdateState =
    "swUpdateState"


typeStartApp : String
typeStartApp =
    "startApp"


typeStartSystem : String
typeStartSystem =
    "startSystem"


typeUpdateOS : String
typeUpdateOS =
    "updateOS"


typeUpdateApp : String
typeUpdateApp =
    "updateApp"


typeSysState : String
typeSysState =
    "sysState"


typeSwUpdateRunning : String
typeSwUpdateRunning =
    "swUpdateRunning"


typeSwUpdateError : String
typeSwUpdateError =
    "swUpdateError"


typeSwUpdatePercComplete : String
typeSwUpdatePercComplete =
    "swUpdatePercComplete"


typeOSVersion : String
typeOSVersion =
    "osVersion"


typeAppVersion : String
typeAppVersion =
    "appVersion"


typeHwVersion : String
typeHwVersion =
    "hwVersion"


typeFirstName : String
typeFirstName =
    "firstName"


typeLastName : String
typeLastName =
    "lastName"


typeEmail : String
typeEmail =
    "email"


typePhone : String
typePhone =
    "phone"


typePass : String
typePass =
    "pass"


typePort : String
typePort =
    "port"


typeBaud : String
typeBaud =
    "baud"


typeID : String
typeID =
    "id"


typeAddress : String
typeAddress =
    "address"


typeModbusIOType : String
typeModbusIOType =
    "modbusIoType"


valueModbusDiscreteInput : String
valueModbusDiscreteInput =
    "modbusDiscreteInput"


valueModbusCoil : String
valueModbusCoil =
    "modbusCoil"


valueModbusInputRegister : String
valueModbusInputRegister =
    "modbusInputRegister"


valueModbusHoldingRegister : String
valueModbusHoldingRegister =
    "modbusHoldingRegister"


typeDataFormat : String
typeDataFormat =
    "dataFormat"


typeDebug : String
typeDebug =
    "debug"


valueUINT16 : String
valueUINT16 =
    "uint16"


valueINT16 : String
valueINT16 =
    "int16"


valueUINT32 : String
valueUINT32 =
    "uint32"


valueINT32 : String
valueINT32 =
    "int32"


valueFLOAT32 : String
valueFLOAT32 =
    "float32"


typeClientServer : String
typeClientServer =
    "clientServer"


valueClient : String
valueClient =
    "client"


valueServer : String
valueServer =
    "server"



-- Point should match data/Point.go


type alias Point =
    { id : String
    , typ : String
    , index : Int
    , time : Time.Posix
    , value : Float
    , text : String
    , min : Float
    , max : Float
    }


empty : Point
empty =
    Point
        ""
        ""
        0
        (Time.millisToPosix 0)
        0
        ""
        0
        0


newValue : String -> String -> Float -> Point
newValue id typ value =
    { id = id
    , typ = typ
    , index = 0
    , time = Time.millisToPosix 0
    , value = value
    , text = ""
    , min = 0
    , max = 0
    }


newText : String -> String -> String -> Point
newText id typ text =
    { id = id
    , typ = typ
    , index = 0
    , time = Time.millisToPosix 0
    , value = 0
    , text = text
    , min = 0
    , max = 0
    }


specialPoints : List String
specialPoints =
    [ typeDescription
    , typeHwVersion
    , typeOSVersion
    , typeAppVersion
    ]


filterSpecialPoints : List Point -> List Point
filterSpecialPoints points =
    List.filter (\p -> not <| List.member p.typ specialPoints) points


encode : Point -> Json.Encode.Value
encode s =
    Json.Encode.object
        [ ( "id", Json.Encode.string <| s.id )
        , ( "type", Json.Encode.string <| s.typ )
        , ( "index", Json.Encode.int <| s.index )
        , ( "time", Iso8601.encode <| s.time )
        , ( "value", Json.Encode.float <| s.value )
        , ( "text", Json.Encode.string <| s.text )
        , ( "min", Json.Encode.float <| s.min )
        , ( "max", Json.Encode.float <| s.max )
        ]


encodeList : List Point -> Json.Encode.Value
encodeList p =
    Json.Encode.list encode p


decode : Decode.Decoder Point
decode =
    Decode.succeed Point
        |> optional "id" Decode.string ""
        |> optional "type" Decode.string ""
        |> optional "index" Decode.int 0
        |> optional "time" Json.Decode.Extra.datetime (Time.millisToPosix 0)
        |> optional "value" Decode.float 0
        |> optional "text" Decode.string ""
        |> optional "min" Decode.float 0
        |> optional "max" Decode.float 0


renderPoint : Point -> String
renderPoint s =
    let
        id =
            if s.id == "" then
                ""

            else
                s.id ++ ": "

        value =
            if s.text /= "" then
                s.text

            else
                Round.round 2 s.value
    in
    id ++ value ++ " (" ++ s.typ ++ ")"


updatePoint : List Point -> Point -> List Point
updatePoint points point =
    case
        List.Extra.findIndex
            (\p ->
                point.id == p.id && point.typ == p.typ && point.index == p.index
            )
            points
    of
        Just index ->
            List.Extra.setAt index point points

        Nothing ->
            point :: points


updatePoints : List Point -> List Point -> List Point
updatePoints points newPoints =
    List.foldr
        (\newPoint updatedPoints -> updatePoint updatedPoints newPoint)
        points
        newPoints


get : List Point -> String -> String -> Int -> Maybe Point
get points id typ index =
    List.Extra.find
        (\p ->
            id == p.id && typ == p.typ && index == p.index
        )
        points


getText : List Point -> String -> String
getText points typ =
    case
        List.Extra.find
            (\p ->
                typ == p.typ
            )
            points
    of
        Just found ->
            found.text

        Nothing ->
            ""


getValue : List Point -> String -> Float
getValue points typ =
    case
        List.Extra.find
            (\p ->
                typ == p.typ
            )
            points
    of
        Just found ->
            found.value

        Nothing ->
            0


getLatest : List Point -> Maybe Point
getLatest points =
    List.foldl
        (\p result ->
            case result of
                Just point ->
                    if Time.posixToMillis p.time > Time.posixToMillis point.time then
                        Just p

                    else
                        Just point

                Nothing ->
                    Just p
        )
        Nothing
        points



-- clearText is used to sanitize points that have number values before saving.
-- the text value is used by the form when editting things like decimal points


blankMajicValue : String
blankMajicValue =
    "123BLANK123"


clearText : List Point -> List Point
clearText points =
    List.map
        (\p ->
            if p.value /= 0 || p.text == blankMajicValue then
                { p | text = "" }

            else
                p
        )
        points
