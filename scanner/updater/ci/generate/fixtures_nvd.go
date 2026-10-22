package main

import (
	schema "github.com/facebookincubator/nvdtools/cveapi/nvd/schema"
)

// nvd.json.zst native fixture relationships, transcribed from the checked-in CI bundle.
// Kept independent from expected results in scanner/e2etests and backend QA.
// Enrichment closure for native fixture CVEs. TestImage compares descriptions
// and CVSS; TestFixtureMatching checks jackson-databind/CVE-2020-24616 (8.1).
// These typed NVD payloads do not themselves create matching vulnerabilities.
var nvdFixtures = []operation{
	{Member: "nvd.json.zst", Updater: "nvd", Enrichments: []enrichmentFixture{
		{Tags: []string{"CVE-2008-6505"}, Payload: nvdValue5},
		{Tags: []string{"CVE-2008-6682"}, Payload: nvdValue11},
		{Tags: []string{"CVE-2010-1870"}, Payload: nvdValue17},
		{Tags: []string{"CVE-2011-1772"}, Payload: nvdValue23},
		{Tags: []string{"CVE-2011-3923"}, Payload: nvdValue31},
		{Tags: []string{"CVE-2011-3374"}, Payload: nvdValue37},
		{Tags: []string{"CVE-2012-0391"}, Payload: nvdValue43},
		{Tags: []string{"CVE-2012-0392"}, Payload: nvdValue49},
		{Tags: []string{"CVE-2012-0393"}, Payload: nvdValue55},
		{Tags: []string{"CVE-2012-0838"}, Payload: nvdValue61},
		{Tags: []string{"CVE-2012-4386"}, Payload: nvdValue64},
		{Tags: []string{"CVE-2012-1592"}, Payload: nvdValue72},
		{Tags: []string{"CVE-2013-1965"}, Payload: nvdValue76},
		{Tags: []string{"CVE-2013-1966"}, Payload: nvdValue79},
		{Tags: []string{"CVE-2013-2115"}, Payload: nvdValue85},
		{Tags: []string{"CVE-2013-2134"}, Payload: nvdValue88},
		{Tags: []string{"CVE-2013-2135"}, Payload: nvdValue91},
		{Tags: []string{"CVE-2013-2248"}, Payload: nvdValue97},
		{Tags: []string{"CVE-2013-2251"}, Payload: nvdValue100},
		{Tags: []string{"CVE-2013-4310"}, Payload: nvdValue102},
		{Tags: []string{"CVE-2013-4316"}, Payload: nvdValue105},
		{Tags: []string{"CVE-2013-6348"}, Payload: nvdValue108},
		{Tags: []string{"CVE-2014-0094"}, Payload: nvdValue111},
		{Tags: []string{"CVE-2014-0112"}, Payload: nvdValue115},
		{Tags: []string{"CVE-2014-0113"}, Payload: nvdValue118},
		{Tags: []string{"CVE-2014-0116"}, Payload: nvdValue124},
		{Tags: []string{"CVE-2014-7809"}, Payload: nvdValue127},
		{Tags: []string{"CVE-2015-1831"}, Payload: nvdValue130},
		{Tags: []string{"CVE-2015-5209"}, Payload: nvdValue136},
		{Tags: []string{"CVE-2015-5169"}, Payload: nvdValue141},
		{Tags: []string{"CVE-2015-2992"}, Payload: nvdValue146},
		{Tags: []string{"CVE-2016-0785"}, Payload: nvdValue154},
		{Tags: []string{"CVE-2016-2162"}, Payload: nvdValue157},
		{Tags: []string{"CVE-2016-4003"}, Payload: nvdValue160},
		{Tags: []string{"CVE-2016-3081"}, Payload: nvdValue166},
		{Tags: []string{"CVE-2016-3082"}, Payload: nvdValue172},
		{Tags: []string{"CVE-2016-3087"}, Payload: nvdValue176},
		{Tags: []string{"CVE-2016-3093"}, Payload: nvdValue184},
		{Tags: []string{"CVE-2016-4438"}, Payload: nvdValue187},
		{Tags: []string{"CVE-2016-4465"}, Payload: nvdValue190},
		{Tags: []string{"CVE-2016-5131"}, Payload: nvdValue196},
		{Tags: []string{"CVE-2016-4436"}, Payload: nvdValue199},
		{Tags: []string{"CVE-2016-9318"}, Payload: nvdValue207},
		{Tags: []string{"CVE-2016-8859"}, Payload: nvdValue210},
		{Tags: []string{"CVE-2016-8738"}, Payload: nvdValue218},
		{Tags: []string{"CVE-2016-4461"}, Payload: nvdValue221},
		{Tags: []string{"CVE-2017-5638"}, Payload: nvdValue225},
		{Tags: []string{"CVE-2017-7186"}, Payload: nvdValue231},
		{Tags: []string{"CVE-2017-5969"}, Payload: nvdValue239},
		{Tags: []string{"CVE-2017-8105"}, Payload: nvdValue242},
		{Tags: []string{"CVE-2017-5029"}, Payload: nvdValue248},
		{Tags: []string{"CVE-2017-8287"}, Payload: nvdValue251},
		{Tags: []string{"CVE-2017-9525"}, Payload: nvdValue259},
		{Tags: []string{"CVE-2017-11164"}, Payload: nvdValue265},
		{Tags: []string{"CVE-2017-7529"}, Payload: nvdValue271},
		{Tags: []string{"CVE-2017-7672"}, Payload: nvdValue274},
		{Tags: []string{"CVE-2017-9787"}, Payload: nvdValue277},
		{Tags: []string{"CVE-2017-9614"}, Payload: nvdValue280},
		{Tags: []string{"CVE-2017-12611"}, Payload: nvdValue283},
		{Tags: []string{"CVE-2017-9804"}, Payload: nvdValue286},
		{Tags: []string{"CVE-2017-15873"}, Payload: nvdValue292},
		{Tags: []string{"CVE-2017-15874"}, Payload: nvdValue298},
		{Tags: []string{"CVE-2017-1000382"}, Payload: nvdValue306},
		{Tags: []string{"CVE-2017-16544"}, Payload: nvdValue309},
		{Tags: []string{"CVE-2017-16231"}, Payload: nvdValue317},
		{Tags: []string{"CVE-2018-6942"}, Payload: nvdValue323},
		{Tags: []string{"CVE-2018-9251"}, Payload: nvdValue329},
		{Tags: []string{"CVE-2018-7160"}, Payload: nvdValue332},
		{Tags: []string{"CVE-2018-11813"}, Payload: nvdValue334},
		{Tags: []string{"CVE-2018-0495"}, Payload: nvdValue342},
		{Tags: []string{"CVE-2018-1152"}, Payload: nvdValue345},
		{Tags: []string{"CVE-2018-1000500"}, Payload: nvdValue349},
		{Tags: []string{"CVE-2018-14048"}, Payload: nvdValue355},
		{Tags: []string{"CVE-2018-14404"}, Payload: nvdValue358},
		{Tags: []string{"CVE-2018-14567"}, Payload: nvdValue361},
		{Tags: []string{"CVE-2018-11776"}, Payload: nvdValue364},
		{Tags: []string{"CVE-2018-0735"}, Payload: nvdValue370},
		{Tags: []string{"CVE-2018-0734"}, Payload: nvdValue373},
		{Tags: []string{"CVE-2018-16843"}, Payload: nvdValue379},
		{Tags: []string{"CVE-2018-16844"}, Payload: nvdValue382},
		{Tags: []string{"CVE-2018-16845"}, Payload: nvdValue390},
		{Tags: []string{"CVE-2018-5407"}, Payload: nvdValue396},
		{Tags: []string{"CVE-2018-20679"}, Payload: nvdValue402},
		{Tags: []string{"CVE-2018-14498"}, Payload: nvdValue405},
		{Tags: []string{"CVE-2018-14550"}, Payload: nvdValue408},
		{Tags: []string{"CVE-2019-5747"}, Payload: nvdValue411},
		{Tags: []string{"CVE-2019-7317"}, Payload: nvdValue417},
		{Tags: []string{"CVE-2019-1543"}, Payload: nvdValue423},
		{Tags: []string{"CVE-2019-9704"}, Payload: nvdValue429},
		{Tags: []string{"CVE-2019-9705"}, Payload: nvdValue432},
		{Tags: []string{"CVE-2019-9706"}, Payload: nvdValue435},
		{Tags: []string{"CVE-2019-11068"}, Payload: nvdValue438},
		{Tags: []string{"CVE-2019-5435"}, Payload: nvdValue443},
		{Tags: []string{"CVE-2019-5436"}, Payload: nvdValue451},
		{Tags: []string{"CVE-2019-12904"}, Payload: nvdValue454},
		{Tags: []string{"CVE-2019-13117"}, Payload: nvdValue460},
		{Tags: []string{"CVE-2019-13118"}, Payload: nvdValue463},
		{Tags: []string{"CVE-2019-14697"}, Payload: nvdValue466},
		{Tags: []string{"CVE-2019-9511"}, Payload: nvdValue469},
		{Tags: []string{"CVE-2019-9513"}, Payload: nvdValue472},
		{Tags: []string{"CVE-2019-9516"}, Payload: nvdValue480},
		{Tags: []string{"CVE-2019-5481"}, Payload: nvdValue482},
		{Tags: []string{"CVE-2019-5482"}, Payload: nvdValue484},
		{Tags: []string{"CVE-2019-13627"}, Payload: nvdValue492},
		{Tags: []string{"CVE-2019-18197"}, Payload: nvdValue500},
		{Tags: []string{"CVE-2019-2201"}, Payload: nvdValue506},
		{Tags: []string{"CVE-2019-1551"}, Payload: nvdValue509},
		{Tags: []string{"CVE-2019-19956"}, Payload: nvdValue512},
		{Tags: []string{"CVE-2019-20372"}, Payload: nvdValue516},
		{Tags: []string{"CVE-2019-20387"}, Payload: nvdValue519},
		{Tags: []string{"CVE-2019-0230"}, Payload: nvdValue522},
		{Tags: []string{"CVE-2019-0233"}, Payload: nvdValue525},
		{Tags: []string{"CVE-2020-8116"}, Payload: nvdValue531},
		{Tags: []string{"CVE-2020-7608"}, Payload: nvdValue536},
		{Tags: []string{"CVE-2020-1967"}, Payload: nvdValue539},
		{Tags: []string{"CVE-2020-9488"}, Payload: nvdValue545},
		{Tags: []string{"CVE-2020-14061"}, Payload: nvdValue548},
		{Tags: []string{"CVE-2020-14062"}, Payload: nvdValue551},
		{Tags: []string{"CVE-2020-14060"}, Payload: nvdValue554},
		{Tags: []string{"CVE-2020-14155"}, Payload: nvdValue559},
		{Tags: []string{"CVE-2020-14195"}, Payload: nvdValue562},
		{Tags: []string{"CVE-2020-15095"}, Payload: nvdValue568},
		{Tags: []string{"CVE-2020-15366"}, Payload: nvdValue574},
		{Tags: []string{"CVE-2020-7699"}, Payload: nvdValue577},
		{Tags: []string{"CVE-2020-24616"}, Payload: nvdValue580},
		{Tags: []string{"CVE-2020-24977"}, Payload: nvdValue588},
		{Tags: []string{"CVE-2020-24750"}, Payload: nvdValue591},
		{Tags: []string{"CVE-2020-8252"}, Payload: nvdValue594},
		{Tags: []string{"CVE-2020-7754"}, Payload: nvdValue598},
		{Tags: []string{"CVE-2020-15999"}, Payload: nvdValue604},
		{Tags: []string{"CVE-2020-7774"}, Payload: nvdValue606},
		{Tags: []string{"CVE-2020-28928"}, Payload: nvdValue609},
		{Tags: []string{"CVE-2020-25649"}, Payload: nvdValue615},
		{Tags: []string{"CVE-2020-1971"}, Payload: nvdValue621},
		{Tags: []string{"CVE-2020-17530"}, Payload: nvdValue624},
		{Tags: []string{"CVE-2020-7788"}, Payload: nvdValue627},
		{Tags: []string{"CVE-2020-8169"}, Payload: nvdValue630},
		{Tags: []string{"CVE-2020-8177"}, Payload: nvdValue633},
		{Tags: []string{"CVE-2020-8231"}, Payload: nvdValue635},
		{Tags: []string{"CVE-2020-8284"}, Payload: nvdValue638},
		{Tags: []string{"CVE-2020-8285"}, Payload: nvdValue641},
		{Tags: []string{"CVE-2020-8286"}, Payload: nvdValue644},
		{Tags: []string{"CVE-2020-35490"}, Payload: nvdValue647},
		{Tags: []string{"CVE-2020-35491"}, Payload: nvdValue650},
		{Tags: []string{"CVE-2020-8265"}, Payload: nvdValue653},
		{Tags: []string{"CVE-2020-8287"}, Payload: nvdValue661},
		{Tags: []string{"CVE-2020-36518"}, Payload: nvdValue664},
		{Tags: []string{"CVE-2021-23840"}, Payload: nvdValue667},
		{Tags: []string{"CVE-2021-23841"}, Payload: nvdValue670},
		{Tags: []string{"CVE-2021-22883"}, Payload: nvdValue673},
		{Tags: []string{"CVE-2021-22884"}, Payload: nvdValue676},
		{Tags: []string{"CVE-2021-28831"}, Payload: nvdValue679},
		{Tags: []string{"CVE-2021-3449"}, Payload: nvdValue682},
		{Tags: []string{"CVE-2021-3450"}, Payload: nvdValue688},
		{Tags: []string{"CVE-2021-22876"}, Payload: nvdValue691},
		{Tags: []string{"CVE-2021-22890"}, Payload: nvdValue694},
		{Tags: []string{"CVE-2021-30139"}, Payload: nvdValue696},
		{Tags: []string{"CVE-2021-26291"}, Payload: nvdValue702},
		{Tags: []string{"CVE-2021-3200"}, Payload: nvdValue708},
		{Tags: []string{"CVE-2021-3520"}, Payload: nvdValue711},
		{Tags: []string{"CVE-2021-33560"}, Payload: nvdValue714},
		{Tags: []string{"CVE-2021-22898"}, Payload: nvdValue722},
		{Tags: []string{"CVE-2021-33910"}, Payload: nvdValue728},
		{Tags: []string{"CVE-2021-36159"}, Payload: nvdValue734},
		{Tags: []string{"CVE-2021-22924"}, Payload: nvdValue737},
		{Tags: []string{"CVE-2021-22925"}, Payload: nvdValue740},
		{Tags: []string{"CVE-2021-3711"}, Payload: nvdValue743},
		{Tags: []string{"CVE-2021-3712"}, Payload: nvdValue749},
		{Tags: []string{"CVE-2021-33928"}, Payload: nvdValue752},
		{Tags: []string{"CVE-2021-33929"}, Payload: nvdValue755},
		{Tags: []string{"CVE-2021-33930"}, Payload: nvdValue758},
		{Tags: []string{"CVE-2021-33938"}, Payload: nvdValue761},
		{Tags: []string{"CVE-2021-40528"}, Payload: nvdValue765},
		{Tags: []string{"CVE-2021-22946"}, Payload: nvdValue768},
		{Tags: []string{"CVE-2021-22947"}, Payload: nvdValue774},
		{Tags: []string{"CVE-2021-22960"}, Payload: nvdValue778},
		{Tags: []string{"CVE-2021-43616"}, Payload: nvdValue781},
		{Tags: []string{"CVE-2021-22959"}, Payload: nvdValue784},
		{Tags: []string{"CVE-2021-45105"}, Payload: nvdValue787},
		{Tags: []string{"CVE-2021-44832"}, Payload: nvdValue795},
		{Tags: []string{"CVE-2021-45960"}, Payload: nvdValue799},
		{Tags: []string{"CVE-2021-46143"}, Payload: nvdValue802},
		{Tags: []string{"CVE-2021-44531"}, Payload: nvdValue805},
		{Tags: []string{"CVE-2021-44532"}, Payload: nvdValue811},
		{Tags: []string{"CVE-2021-44533"}, Payload: nvdValue814},
		{Tags: []string{"CVE-2021-31805"}, Payload: nvdValue817},
		{Tags: []string{"CVE-2021-41411"}, Payload: nvdValue820},
		{Tags: []string{"CVE-2022-22822"}, Payload: nvdValue822},
		{Tags: []string{"CVE-2022-22823"}, Payload: nvdValue824},
		{Tags: []string{"CVE-2022-22824"}, Payload: nvdValue826},
		{Tags: []string{"CVE-2022-22825"}, Payload: nvdValue828},
		{Tags: []string{"CVE-2022-22826"}, Payload: nvdValue830},
		{Tags: []string{"CVE-2022-22827"}, Payload: nvdValue832},
		{Tags: []string{"CVE-2022-23852"}, Payload: nvdValue835},
		{Tags: []string{"CVE-2022-23990"}, Payload: nvdValue837},
		{Tags: []string{"CVE-2022-25235"}, Payload: nvdValue840},
		{Tags: []string{"CVE-2022-25236"}, Payload: nvdValue843},
		{Tags: []string{"CVE-2022-25313"}, Payload: nvdValue846},
		{Tags: []string{"CVE-2022-25314"}, Payload: nvdValue848},
		{Tags: []string{"CVE-2022-25315"}, Payload: nvdValue850},
		{Tags: []string{"CVE-2022-21824"}, Payload: nvdValue856},
		{Tags: []string{"CVE-2022-0778"}, Payload: nvdValue859},
		{Tags: []string{"CVE-2022-22963"}, Payload: nvdValue862},
		{Tags: []string{"CVE-2022-22965"}, Payload: nvdValue865},
		{Tags: []string{"CVE-2022-27261"}, Payload: nvdValue869},
		{Tags: []string{"CVE-2022-1292"}, Payload: nvdValue875},
		{Tags: []string{"CVE-2022-1343"}, Payload: nvdValue879},
		{Tags: []string{"CVE-2022-1434"}, Payload: nvdValue882},
		{Tags: []string{"CVE-2022-1473"}, Payload: nvdValue885},
		{Tags: []string{"CVE-2022-29885"}, Payload: nvdValue888},
		{Tags: []string{"CVE-2022-1650"}, Payload: nvdValue894},
		{Tags: []string{"CVE-2022-30065"}, Payload: nvdValue897},
		{Tags: []string{"CVE-2022-1785"}, Payload: nvdValue899},
		{Tags: []string{"CVE-2022-22978"}, Payload: nvdValue902},
		{Tags: []string{"CVE-2022-22576"}, Payload: nvdValue910},
		{Tags: []string{"CVE-2022-1897"}, Payload: nvdValue912},
		{Tags: []string{"CVE-2022-1927"}, Payload: nvdValue914},
		{Tags: []string{"CVE-2022-27774"}, Payload: nvdValue922},
		{Tags: []string{"CVE-2022-27775"}, Payload: nvdValue925},
		{Tags: []string{"CVE-2022-27776"}, Payload: nvdValue931},
		{Tags: []string{"CVE-2022-27781"}, Payload: nvdValue934},
		{Tags: []string{"CVE-2022-27782"}, Payload: nvdValue937},
		{Tags: []string{"CVE-2022-2068"}, Payload: nvdValue940},
		{Tags: []string{"CVE-2022-34176"}, Payload: nvdValue948},
		{Tags: []string{"CVE-2022-34177"}, Payload: nvdValue951},
		{Tags: []string{"CVE-2022-2097"}, Payload: nvdValue954},
		{Tags: []string{"CVE-2022-32206"}, Payload: nvdValue957},
		{Tags: []string{"CVE-2022-32208"}, Payload: nvdValue960},
		{Tags: []string{"CVE-2022-32212"}, Payload: nvdValue964},
		{Tags: []string{"CVE-2022-32213"}, Payload: nvdValue968},
		{Tags: []string{"CVE-2022-32214"}, Payload: nvdValue971},
		{Tags: []string{"CVE-2022-32215"}, Payload: nvdValue974},
		{Tags: []string{"CVE-2022-32222"}, Payload: nvdValue978},
		{Tags: []string{"CVE-2022-32223"}, Payload: nvdValue982},
		{Tags: []string{"CVE-2022-40674"}, Payload: nvdValue984},
		{Tags: []string{"CVE-2022-35252"}, Payload: nvdValue990},
		{Tags: []string{"CVE-2022-3358"}, Payload: nvdValue994},
		{Tags: []string{"CVE-2022-43680"}, Payload: nvdValue998},
		{Tags: []string{"CVE-2022-3602"}, Payload: nvdValue1001},
		{Tags: []string{"CVE-2022-3786"}, Payload: nvdValue1004},
		{Tags: []string{"CVE-2022-32221"}, Payload: nvdValue1008},
		{Tags: []string{"CVE-2022-35255"}, Payload: nvdValue1012},
		{Tags: []string{"CVE-2022-35256"}, Payload: nvdValue1015},
		{Tags: []string{"CVE-2022-43548"}, Payload: nvdValue1018},
		{Tags: []string{"CVE-2022-4304"}, Payload: nvdValue1022},
		{Tags: []string{"CVE-2022-4450"}, Payload: nvdValue1025},
		{Tags: []string{"CVE-2022-43552"}, Payload: nvdValue1029},
		{Tags: []string{"CVE-2022-3219"}, Payload: nvdValue1035},
		{Tags: []string{"CVE-2022-25883"}, Payload: nvdValue1038},
		{Tags: []string{"CVE-2023-0687"}, Payload: nvdValue1044},
		{Tags: []string{"CVE-2023-0215"}, Payload: nvdValue1047},
		{Tags: []string{"CVE-2023-0286"}, Payload: nvdValue1051},
		{Tags: []string{"CVE-2023-23936"}, Payload: nvdValue1057},
		{Tags: []string{"CVE-2023-23916"}, Payload: nvdValue1061},
		{Tags: []string{"CVE-2023-23918"}, Payload: nvdValue1064},
		{Tags: []string{"CVE-2023-23919"}, Payload: nvdValue1067},
		{Tags: []string{"CVE-2023-23920"}, Payload: nvdValue1073},
		{Tags: []string{"CVE-2023-28708"}, Payload: nvdValue1079},
		{Tags: []string{"CVE-2023-0464"}, Payload: nvdValue1082},
		{Tags: []string{"CVE-2023-0465"}, Payload: nvdValue1085},
		{Tags: []string{"CVE-2023-0466"}, Payload: nvdValue1088},
		{Tags: []string{"CVE-2023-27533"}, Payload: nvdValue1092},
		{Tags: []string{"CVE-2023-27534"}, Payload: nvdValue1096},
		{Tags: []string{"CVE-2023-27535"}, Payload: nvdValue1099},
		{Tags: []string{"CVE-2023-27536"}, Payload: nvdValue1102},
		{Tags: []string{"CVE-2023-27538"}, Payload: nvdValue1108},
		{Tags: []string{"CVE-2023-28321"}, Payload: nvdValue1112},
		{Tags: []string{"CVE-2023-28322"}, Payload: nvdValue1116},
		{Tags: []string{"CVE-2023-2650"}, Payload: nvdValue1119},
		{Tags: []string{"CVE-2023-34149"}, Payload: nvdValue1123},
		{Tags: []string{"CVE-2023-34396"}, Payload: nvdValue1126},
		{Tags: []string{"CVE-2023-3138"}, Payload: nvdValue1129},
		{Tags: []string{"CVE-2023-30589"}, Payload: nvdValue1133},
		{Tags: []string{"CVE-2023-35945"}, Payload: nvdValue1136},
		{Tags: []string{"CVE-2023-3446"}, Payload: nvdValue1140},
		{Tags: []string{"CVE-2023-3817"}, Payload: nvdValue1143},
		{Tags: []string{"CVE-2023-32006"}, Payload: nvdValue1146},
		{Tags: []string{"CVE-2023-32002"}, Payload: nvdValue1149},
		{Tags: []string{"CVE-2023-32559"}, Payload: nvdValue1155},
		{Tags: []string{"CVE-2023-4813"}, Payload: nvdValue1158},
		{Tags: []string{"CVE-2023-4527"}, Payload: nvdValue1164},
		{Tags: []string{"CVE-2023-4806"}, Payload: nvdValue1167},
		{Tags: []string{"CVE-2023-5156"}, Payload: nvdValue1170},
		{Tags: []string{"CVE-2023-4911"}, Payload: nvdValue1174},
		{Tags: []string{"CVE-2023-44487"}, Payload: nvdValue1177},
		{Tags: []string{"CVE-2023-38545"}, Payload: nvdValue1180},
		{Tags: []string{"CVE-2023-38546"}, Payload: nvdValue1184},
		{Tags: []string{"CVE-2023-38552"}, Payload: nvdValue1187},
		{Tags: []string{"CVE-2023-5678"}, Payload: nvdValue1190},
		{Tags: []string{"CVE-2023-30581"}, Payload: nvdValue1193},
		{Tags: []string{"CVE-2023-30585"}, Payload: nvdValue1196},
		{Tags: []string{"CVE-2023-30588"}, Payload: nvdValue1199},
		{Tags: []string{"CVE-2023-30590"}, Payload: nvdValue1202},
		{Tags: []string{"CVE-2023-41835"}, Payload: nvdValue1205},
		{Tags: []string{"CVE-2023-46218"}, Payload: nvdValue1208},
		{Tags: []string{"CVE-2023-50164"}, Payload: nvdValue1211},
		{Tags: []string{"CVE-2023-48795"}, Payload: nvdValue1214},
		{Tags: []string{"CVE-2023-7008"}, Payload: nvdValue1217},
		{Tags: []string{"CVE-2023-7104"}, Payload: nvdValue1221},
		{Tags: []string{"CVE-2023-6246"}, Payload: nvdValue1224},
		{Tags: []string{"CVE-2023-6779"}, Payload: nvdValue1227},
		{Tags: []string{"CVE-2023-52425"}, Payload: nvdValue1230},
		{Tags: []string{"CVE-2023-39333"}, Payload: nvdValue1234},
		{Tags: []string{"CVE-2024-0727"}, Payload: nvdValue1238},
		{Tags: []string{"CVE-2024-28757"}, Payload: nvdValue1241},
		{Tags: []string{"CVE-2024-2398"}, Payload: nvdValue1244},
		{Tags: []string{"CVE-2024-2511"}, Payload: nvdValue1247},
		{Tags: []string{"CVE-2024-3566"}, Payload: nvdValue1250},
		{Tags: []string{"CVE-2024-2961"}, Payload: nvdValue1253},
		{Tags: []string{"CVE-2024-33599"}, Payload: nvdValue1256},
		{Tags: []string{"CVE-2024-33601"}, Payload: nvdValue1259},
		{Tags: []string{"CVE-2024-33602"}, Payload: nvdValue1262},
		{Tags: []string{"CVE-2024-5535"}, Payload: nvdValue1265},
		{Tags: []string{"CVE-2024-7264"}, Payload: nvdValue1268},
		{Tags: []string{"CVE-2024-45490"}, Payload: nvdValue1270},
		{Tags: []string{"CVE-2024-45491"}, Payload: nvdValue1273},
		{Tags: []string{"CVE-2024-45492"}, Payload: nvdValue1276},
		{Tags: []string{"CVE-2024-8096"}, Payload: nvdValue1279},
		{Tags: []string{"CVE-2024-9143"}, Payload: nvdValue1282},
		{Tags: []string{"CVE-2024-50602"}, Payload: nvdValue1285},
		{Tags: []string{"CVE-2024-4741"}, Payload: nvdValue1288},
		{Tags: []string{"CVE-2024-11053"}, Payload: nvdValue1291},
		{Tags: []string{"CVE-2024-53677"}, Payload: nvdValue1294},
		{Tags: []string{"CVE-2024-45337"}, Payload: nvdValue1297},
		{Tags: []string{"CVE-2024-13176"}, Payload: nvdValue1300},
		{Tags: []string{"CVE-2024-8176"}, Payload: nvdValue1303},
		{Tags: []string{"CVE-2025-23083"}, Payload: nvdValue1306},
		{Tags: []string{"CVE-2025-23084"}, Payload: nvdValue1309},
		{Tags: []string{"CVE-2025-23085"}, Payload: nvdValue1312},
		{Tags: []string{"CVE-2025-22869"}, Payload: nvdValue1315},
		{Tags: []string{"CVE-2025-24813"}, Payload: nvdValue1318},
		{Tags: []string{"CVE-2025-4802"}, Payload: nvdValue1321},
		{Tags: []string{"CVE-2025-6069"}, Payload: nvdValue1324},
		{Tags: []string{"CVE-2025-8058"}, Payload: nvdValue1327},
		{Tags: []string{"CVE-2025-59375"}, Payload: nvdValue1330},
		{Tags: []string{"CVE-2025-9230"}, Payload: nvdValue1333},
		{Tags: []string{"CVE-2025-9232"}, Payload: nvdValue1336},
		{Tags: []string{"CVE-2025-8291"}, Payload: nvdValue1339},
		{Tags: []string{"CVE-2025-6075"}, Payload: nvdValue1343},
		{Tags: []string{"CVE-2025-66382"}, Payload: nvdValue1346},
		{Tags: []string{"CVE-2025-64775"}, Payload: nvdValue1349},
		{Tags: []string{"CVE-2025-13837"}, Payload: nvdValue1352},
		{Tags: []string{"CVE-2025-12084"}, Payload: nvdValue1355},
		{Tags: []string{"CVE-2025-66675"}, Payload: nvdValue1358},
		{Tags: []string{"CVE-2025-14017"}, Payload: nvdValue1361},
		{Tags: []string{"CVE-2025-14524"}, Payload: nvdValue1364},
		{Tags: []string{"CVE-2025-15079"}, Payload: nvdValue1367},
		{Tags: []string{"CVE-2025-15224"}, Payload: nvdValue1370},
		{Tags: []string{"CVE-2025-68493"}, Payload: nvdValue1376},
		{Tags: []string{"CVE-2025-15281"}, Payload: nvdValue1379},
		{Tags: []string{"CVE-2025-11468"}, Payload: nvdValue1382},
		{Tags: []string{"CVE-2025-15282"}, Payload: nvdValue1385},
		{Tags: []string{"CVE-2025-11187"}, Payload: nvdValue1388},
		{Tags: []string{"CVE-2025-15467"}, Payload: nvdValue1391},
		{Tags: []string{"CVE-2025-15468"}, Payload: nvdValue1394},
		{Tags: []string{"CVE-2025-15469"}, Payload: nvdValue1397},
		{Tags: []string{"CVE-2025-66199"}, Payload: nvdValue1400},
		{Tags: []string{"CVE-2025-68160"}, Payload: nvdValue1403},
		{Tags: []string{"CVE-2025-69418"}, Payload: nvdValue1406},
		{Tags: []string{"CVE-2025-69419"}, Payload: nvdValue1409},
		{Tags: []string{"CVE-2025-69420"}, Payload: nvdValue1412},
		{Tags: []string{"CVE-2025-69421"}, Payload: nvdValue1415},
		{Tags: []string{"CVE-2026-0861"}, Payload: nvdValue1418},
		{Tags: []string{"CVE-2026-0915"}, Payload: nvdValue1421},
		{Tags: []string{"CVE-2026-0672"}, Payload: nvdValue1424},
		{Tags: []string{"CVE-2026-0865"}, Payload: nvdValue1426},
		{Tags: []string{"CVE-2026-24515"}, Payload: nvdValue1431},
		{Tags: []string{"CVE-2026-1299"}, Payload: nvdValue1434},
		{Tags: []string{"CVE-2026-22795"}, Payload: nvdValue1437},
		{Tags: []string{"CVE-2026-22796"}, Payload: nvdValue1440},
		{Tags: []string{"CVE-2026-25210"}, Payload: nvdValue1443},
		{Tags: []string{"CVE-2026-2297"}, Payload: nvdValue1446},
		{Tags: []string{"CVE-2026-1965"}, Payload: nvdValue1449},
		{Tags: []string{"CVE-2026-3783"}, Payload: nvdValue1452},
		{Tags: []string{"CVE-2026-3784"}, Payload: nvdValue1455},
		{Tags: []string{"CVE-2026-32776"}, Payload: nvdValue1457},
		{Tags: []string{"CVE-2026-32777"}, Payload: nvdValue1459},
		{Tags: []string{"CVE-2026-32778"}, Payload: nvdValue1462},
		{Tags: []string{"CVE-2026-3644"}, Payload: nvdValue1465},
		{Tags: []string{"CVE-2026-4224"}, Payload: nvdValue1468},
		{Tags: []string{"CVE-2026-4519"}, Payload: nvdValue1474},
		{Tags: []string{"CVE-2026-4437"}, Payload: nvdValue1477},
		{Tags: []string{"CVE-2026-4046"}, Payload: nvdValue1480},
		{Tags: []string{"CVE-2026-28387"}, Payload: nvdValue1483},
		{Tags: []string{"CVE-2026-28388"}, Payload: nvdValue1486},
		{Tags: []string{"CVE-2026-28389"}, Payload: nvdValue1489},
		{Tags: []string{"CVE-2026-28390"}, Payload: nvdValue1492},
		{Tags: []string{"CVE-2026-4786"}, Payload: nvdValue1495},
		{Tags: []string{"CVE-2026-41080"}, Payload: nvdValue1497},
		{Tags: []string{"CVE-2026-45186"}, Payload: nvdValue1500},
		{Tags: []string{"CVE-2026-4873"}, Payload: nvdValue1503},
		{Tags: []string{"CVE-2026-5545"}, Payload: nvdValue1509},
		{Tags: []string{"CVE-2026-5773"}, Payload: nvdValue1512},
		{Tags: []string{"CVE-2026-6253"}, Payload: nvdValue1515},
		{Tags: []string{"CVE-2026-6429"}, Payload: nvdValue1518},
		{Tags: []string{"CVE-2026-7168"}, Payload: nvdValue1522},
		{Tags: []string{"CVE-2026-46595"}, Payload: nvdValue1525},
		{Tags: []string{"CVE-2026-50219"}, Payload: nvdValue1531},
		{Tags: []string{"CVE-2026-34180"}, Payload: nvdValue1534},
		{Tags: []string{"CVE-2026-42766"}, Payload: nvdValue1537},
		{Tags: []string{"CVE-2026-45447"}, Payload: nvdValue1540},
		{Tags: []string{"CVE-2026-7383"}, Payload: nvdValue1543},
		{Tags: []string{"CVE-2026-9076"}, Payload: nvdValue1546},
		{Tags: []string{"CVE-2026-56131"}, Payload: nvdValue1549},
		{Tags: []string{"CVE-2026-56132"}, Payload: nvdValue1555},
		{Tags: []string{"CVE-2026-56403"}, Payload: nvdValue1557},
		{Tags: []string{"CVE-2026-56404"}, Payload: nvdValue1559},
		{Tags: []string{"CVE-2026-56405"}, Payload: nvdValue1561},
		{Tags: []string{"CVE-2026-56406"}, Payload: nvdValue1564},
		{Tags: []string{"CVE-2026-56407"}, Payload: nvdValue1566},
		{Tags: []string{"CVE-2026-56408"}, Payload: nvdValue1568},
		{Tags: []string{"CVE-2026-56409"}, Payload: nvdValue1570},
		{Tags: []string{"CVE-2026-56410"}, Payload: nvdValue1572},
		{Tags: []string{"CVE-2026-56411"}, Payload: nvdValue1574},
		{Tags: []string{"CVE-2026-56412"}, Payload: nvdValue1577},
		{Tags: []string{"CVE-2026-11856"}, Payload: nvdValue1580},
		{Tags: []string{"CVE-2026-8286"}, Payload: nvdValue1583},
		{Tags: []string{"CVE-2026-8458"}, Payload: nvdValue1586},
		{Tags: []string{"CVE-2026-8924"}, Payload: nvdValue1589},
		{Tags: []string{"CVE-2026-8927"}, Payload: nvdValue1592},
		{Tags: []string{"CVE-2026-8932"}, Payload: nvdValue1595},
		{Tags: []string{"CVE-2026-72522"}, Payload: nvdValue1598},
		{Tags: []string{"CVE-2026-66046"}, Payload: nvdValue1601},
		{Tags: []string{"CVE-2026-76957"}, Payload: nvdValue1604},
		{Tags: []string{"CVE-2026-76641"}, Payload: nvdValue1607},
		{Tags: []string{"CVE-2026-54874"}, Payload: nvdValue1610},
		{Tags: []string{"CVE-2026-63072"}, Payload: nvdValue1613},
		{Tags: []string{"CVE-2026-18924"}, Payload: nvdValue1616},
		{Tags: []string{"CVE-2026-19931"}, Payload: nvdValue1619},
		{Tags: []string{"CVE-2026-80230"}, Payload: nvdValue1622},
		{Tags: []string{"CVE-2026-82209"}, Payload: nvdValue1625},
		{Tags: []string{"CVE-2026-93990"}, Payload: nvdValue1628},
	}},
}
var nvdValue0 = "Multiple directory traversal vulnerabilities in Apache Struts 2.0.x before 2.0.12 and 2.1.x before 2.1.3 allow remote attackers to read arbitrary files via a ..%252f (encoded dot dot slash) in a URI with a /struts/ path, related to (1) FilterDispatcher in 2.0.x and (2) DefaultStaticContentLoader in 2.1.x."

var nvdValue1 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue0}

var nvdValue2 = schema.CVSSV20{BaseScore: 5, VectorString: "AV:N/AC:L/Au:N/C:P/I:N/A:N", Version: "2.0"}

var nvdValue3 = schema.CVEAPIJSON20CVSSV2{CvssData: &nvdValue2}

var nvdValue4 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue3}}

var nvdValue5 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1}, ID: "CVE-2008-6505", LastModified: "2026-06-16T23:02:21.047", Metrics: &nvdValue4, Published: "2009-03-23T14:19:12.453"}

var nvdValue6 = "Multiple cross-site scripting (XSS) vulnerabilities in Apache Struts 2.0.x before 2.0.11.1 and 2.1.x before 2.1.1 allow remote attackers to inject arbitrary web script or HTML via vectors associated with improper handling of (1) \" (double quote) characters in the href attribute of an s:a tag and (2) parameters in the action attribute of an s:url tag."

var nvdValue7 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue6}

var nvdValue8 = schema.CVSSV20{BaseScore: 4.3, VectorString: "AV:N/AC:M/Au:N/C:N/I:P/A:N", Version: "2.0"}

var nvdValue9 = schema.CVEAPIJSON20CVSSV2{CvssData: &nvdValue8}

var nvdValue10 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue9}}

var nvdValue11 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue7}, ID: "CVE-2008-6682", LastModified: "2026-06-16T23:02:45.853", Metrics: &nvdValue10, Published: "2009-04-09T15:08:35.547"}

var nvdValue12 = "The OGNL extensive expression evaluation capability in XWork in Struts 2.0.0 through 2.1.8.1, as used in Atlassian Fisheye, Crucible, and possibly other products, uses a permissive whitelist, which allows remote attackers to modify server-side context objects and bypass the \"#\" protection mechanism in ParameterInterceptors via the (1) #context, (2) #_memberAccess, (3) #root, (4) #this, (5) #_typeResolver, (6) #_classResolver, (7) #_traceEvaluations, (8) #_lastEvaluation, (9) #_keepLastEvaluation, and possibly other OGNL context variables, a different vulnerability than CVE-2008-6504."

var nvdValue13 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue12}

var nvdValue14 = schema.CVSSV20{BaseScore: 5, VectorString: "AV:N/AC:L/Au:N/C:N/I:P/A:N", Version: "2.0"}

var nvdValue15 = schema.CVEAPIJSON20CVSSV2{CvssData: &nvdValue14}

var nvdValue16 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue15}}

var nvdValue17 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue13}, ID: "CVE-2010-1870", LastModified: "2026-06-16T23:19:29.683", Metrics: &nvdValue16, Published: "2010-08-17T20:00:03.407"}

var nvdValue18 = "Multiple cross-site scripting (XSS) vulnerabilities in XWork in Apache Struts 2.x before 2.2.3, and OpenSymphony XWork in OpenSymphony WebWork, allow remote attackers to inject arbitrary web script or HTML via vectors involving (1) an action name, (2) the action attribute of an s:submit element, or (3) the method attribute of an s:submit element."

var nvdValue19 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue18}

var nvdValue20 = schema.CVSSV20{BaseScore: 2.6, VectorString: "AV:N/AC:H/Au:N/C:N/I:P/A:N", Version: "2.0"}

var nvdValue21 = schema.CVEAPIJSON20CVSSV2{CvssData: &nvdValue20}

var nvdValue22 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue21}}

var nvdValue23 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue19}, ID: "CVE-2011-1772", LastModified: "2026-06-16T23:29:59.930", Metrics: &nvdValue22, Published: "2011-05-13T17:05:44.267"}

var nvdValue24 = "Apache Struts before 2.3.1.2 allows remote attackers to bypass security protections in the ParameterInterceptor class and execute arbitrary commands."

var nvdValue25 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue24}

var nvdValue26 = schema.CVSSV20{BaseScore: 7.5, VectorString: "AV:N/AC:L/Au:N/C:P/I:P/A:P", Version: "2.0"}

var nvdValue27 = schema.CVEAPIJSON20CVSSV2{CvssData: &nvdValue26}

var nvdValue28 = schema.CVSSV31{BaseScore: 9.8, VectorString: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H", Version: "3.1"}

var nvdValue29 = schema.CVEAPIJSON20CVSSV31{CvssData: &nvdValue28}

var nvdValue30 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue27}, CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue29}}

var nvdValue31 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue25}, ID: "CVE-2011-3923", LastModified: "2026-06-16T23:34:09.667", Metrics: &nvdValue30, Published: "2019-11-01T14:15:10.877"}

var nvdValue32 = "It was found that apt-key in apt, all versions, do not correctly validate gpg keys with the master keyring, leading to a potential man-in-the-middle attack."

var nvdValue33 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue32}

var nvdValue34 = schema.CVSSV31{BaseScore: 3.7, VectorString: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:N/I:L/A:N", Version: "3.1"}

var nvdValue35 = schema.CVEAPIJSON20CVSSV31{CvssData: &nvdValue34}

var nvdValue36 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue9}, CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue35}}

var nvdValue37 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue33}, ID: "CVE-2011-3374", LastModified: "2026-06-16T23:33:10.763", Metrics: &nvdValue36, Published: "2019-11-26T00:15:11.030"}

var nvdValue38 = "The ExceptionDelegator component in Apache Struts before 2.2.3.1 interprets parameter values as OGNL expressions during certain exception handling for mismatched data types of properties, which allows remote attackers to execute arbitrary Java code via a crafted parameter."

var nvdValue39 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue38}

var nvdValue40 = schema.CVSSV20{BaseScore: 9.3, VectorString: "AV:N/AC:M/Au:N/C:C/I:C/A:C", Version: "2.0"}

var nvdValue41 = schema.CVEAPIJSON20CVSSV2{CvssData: &nvdValue40}

var nvdValue42 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue41}, CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue29}}

var nvdValue43 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue39}, ID: "CVE-2012-0391", LastModified: "2026-06-16T23:37:12.150", Metrics: &nvdValue42, Published: "2012-01-08T15:55:01.217"}

var nvdValue44 = "The CookieInterceptor component in Apache Struts before 2.3.1.1 does not use the parameter-name whitelist, which allows remote attackers to execute arbitrary commands via a crafted HTTP Cookie header that triggers Java code execution through a static method."

var nvdValue45 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue44}

var nvdValue46 = schema.CVSSV20{BaseScore: 6.8, VectorString: "AV:N/AC:M/Au:N/C:P/I:P/A:P", Version: "2.0"}

var nvdValue47 = schema.CVEAPIJSON20CVSSV2{CvssData: &nvdValue46}

var nvdValue48 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue47}}

var nvdValue49 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue45}, ID: "CVE-2012-0392", LastModified: "2026-06-16T23:37:12.347", Metrics: &nvdValue48, Published: "2012-01-08T15:55:01.373"}

var nvdValue50 = "The ParameterInterceptor component in Apache Struts before 2.3.1.1 does not prevent access to public constructors, which allows remote attackers to create or overwrite arbitrary files via a crafted parameter that triggers the creation of a Java object."

var nvdValue51 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue50}

var nvdValue52 = schema.CVSSV20{BaseScore: 6.4, VectorString: "AV:N/AC:L/Au:N/C:N/I:P/A:P", Version: "2.0"}

var nvdValue53 = schema.CVEAPIJSON20CVSSV2{CvssData: &nvdValue52}

var nvdValue54 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue53}}

var nvdValue55 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue51}, ID: "CVE-2012-0393", LastModified: "2026-06-16T23:37:12.470", Metrics: &nvdValue54, Published: "2012-01-08T15:55:01.420"}

var nvdValue56 = "Apache Struts 2 before 2.2.3.1 evaluates a string as an OGNL expression during the handling of a conversion error, which allows remote attackers to modify run-time data values, and consequently execute arbitrary code, via invalid input to a field."

var nvdValue57 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue56}

var nvdValue58 = schema.CVSSV20{BaseScore: 10, VectorString: "AV:N/AC:L/Au:N/C:C/I:C/A:C", Version: "2.0"}

var nvdValue59 = schema.CVEAPIJSON20CVSSV2{CvssData: &nvdValue58}

var nvdValue60 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue59}}

var nvdValue61 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue57}, ID: "CVE-2012-0838", LastModified: "2026-06-16T23:38:21.580", Metrics: &nvdValue60, Published: "2012-03-02T22:55:01.337"}

var nvdValue62 = "The token check mechanism in Apache Struts 2.0.0 through 2.3.4 does not properly validate the token name configuration parameter, which allows remote attackers to perform cross-site request forgery (CSRF) attacks by setting the token name configuration parameter to a session attribute."

var nvdValue63 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue62}

var nvdValue64 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue63}, ID: "CVE-2012-4386", LastModified: "2026-06-16T23:44:55.210", Metrics: &nvdValue48, Published: "2012-09-05T23:55:02.663"}

var nvdValue65 = "A local code execution issue exists in Apache Struts2 when processing malformed XSLT files, which could let a malicious user upload and execute arbitrary files."

var nvdValue66 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue65}

var nvdValue67 = schema.CVSSV20{BaseScore: 6.5, VectorString: "AV:N/AC:L/Au:S/C:P/I:P/A:P", Version: "2.0"}

var nvdValue68 = schema.CVEAPIJSON20CVSSV2{CvssData: &nvdValue67}

var nvdValue69 = schema.CVSSV31{BaseScore: 8.8, VectorString: "CVSS:3.1/AV:N/AC:L/PR:L/UI:N/S:U/C:H/I:H/A:H", Version: "3.1"}

var nvdValue70 = schema.CVEAPIJSON20CVSSV31{CvssData: &nvdValue69}

var nvdValue71 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue68}, CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue70}}

var nvdValue72 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue66}, ID: "CVE-2012-1592", LastModified: "2026-06-16T23:39:49.353", Metrics: &nvdValue71, Published: "2019-12-05T21:15:11.427"}

var nvdValue73 = "Apache Struts Showcase App 2.0.0 through 2.3.13, as used in Struts 2 before 2.3.14.3, allows remote attackers to execute arbitrary OGNL code via a crafted parameter name that is not properly handled when invoking a redirect."

var nvdValue74 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue73}

var nvdValue75 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue41}}

var nvdValue76 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue74}, ID: "CVE-2013-1965", LastModified: "2026-06-16T23:52:29.303", Metrics: &nvdValue75, Published: "2013-07-10T19:55:04.683"}

var nvdValue77 = "Apache Struts 2 before 2.3.14.2 allows remote attackers to execute arbitrary OGNL code via a crafted request that is not properly handled when using the includeParams attribute in the (1) URL or (2) A tag."

var nvdValue78 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue77}

var nvdValue79 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue78}, ID: "CVE-2013-1966", LastModified: "2026-06-16T23:52:29.403", Metrics: &nvdValue75, Published: "2013-07-10T19:55:04.713"}

var nvdValue80 = "Apache Struts 2 before 2.3.14.2 allows remote attackers to execute arbitrary OGNL code via a crafted request that is not properly handled when using the includeParams attribute in the (1) URL or (2) A tag. NOTE: this issue is due to an incomplete fix for CVE-2013-1966."

var nvdValue81 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue80}

var nvdValue82 = schema.CVSSV31{BaseScore: 8.1, VectorString: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:H/I:H/A:H", Version: "3.1"}

var nvdValue83 = schema.CVEAPIJSON20CVSSV31{CvssData: &nvdValue82}

var nvdValue84 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue41}, CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue83}}

var nvdValue85 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue81}, ID: "CVE-2013-2115", LastModified: "2026-06-16T23:52:46.440", Metrics: &nvdValue84, Published: "2013-07-10T19:55:04.770"}

var nvdValue86 = "Apache Struts 2 before 2.3.14.3 allows remote attackers to execute arbitrary OGNL code via a request with a crafted action name that is not properly handled during wildcard matching, a different vulnerability than CVE-2013-2135."

var nvdValue87 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue86}

var nvdValue88 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue87}, ID: "CVE-2013-2134", LastModified: "2026-06-16T23:52:48.597", Metrics: &nvdValue75, Published: "2013-07-16T18:55:01.380"}

var nvdValue89 = "Apache Struts 2 before 2.3.14.3 allows remote attackers to execute arbitrary OGNL code via a request with a crafted value that contains both \"${}\" and \"%{}\" sequences, which causes the OGNL code to be evaluated twice."

var nvdValue90 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue89}

var nvdValue91 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue90}, ID: "CVE-2013-2135", LastModified: "2026-06-16T23:52:48.707", Metrics: &nvdValue75, Published: "2013-07-16T18:55:01.403"}

var nvdValue92 = "Multiple open redirect vulnerabilities in Apache Struts 2.0.0 through 2.3.15 allow remote attackers to redirect users to arbitrary web sites and conduct phishing attacks via a URL in a parameter using the (1) redirect: or (2) redirectAction: prefix."

var nvdValue93 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue92}

var nvdValue94 = schema.CVSSV20{BaseScore: 5.8, VectorString: "AV:N/AC:M/Au:N/C:P/I:P/A:N", Version: "2.0"}

var nvdValue95 = schema.CVEAPIJSON20CVSSV2{CvssData: &nvdValue94}

var nvdValue96 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue95}}

var nvdValue97 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue93}, ID: "CVE-2013-2248", LastModified: "2026-06-16T23:53:01.037", Metrics: &nvdValue96, Published: "2013-07-20T03:37:30.717"}

var nvdValue98 = "Apache Struts 2.0.0 through 2.3.15 allows remote attackers to execute arbitrary OGNL expressions via a parameter with a crafted (1) action:, (2) redirect:, or (3) redirectAction: prefix."

var nvdValue99 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue98}

var nvdValue100 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue99}, ID: "CVE-2013-2251", LastModified: "2026-06-16T23:53:01.530", Metrics: &nvdValue42, Published: "2013-07-20T03:37:30.737"}

var nvdValue101 = schema.CVEAPIJSON20LangString{Lang: "en", Value: "Apache Struts 2.0.0 through 2.3.15.1 allows remote attackers to bypass access controls via a crafted action: prefix."}

var nvdValue102 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue101}, ID: "CVE-2013-4310", LastModified: "2026-06-16T23:57:00.163", Metrics: &nvdValue96, Published: "2013-09-30T21:55:09.487"}

var nvdValue103 = "Apache Struts 2.0.0 through 2.3.15.1 enables Dynamic Method Invocation by default, which has unknown impact and attack vectors."

var nvdValue104 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue103}

var nvdValue105 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue104}, ID: "CVE-2013-4316", LastModified: "2026-06-16T23:57:00.900", Metrics: &nvdValue60, Published: "2013-09-30T21:55:09.630"}

var nvdValue106 = "Multiple cross-site scripting (XSS) vulnerabilities in Apache Struts 2.3.15.3 allow remote attackers to inject arbitrary web script or HTML via the namespace parameter to (1) actionNames.action and (2) showConfig.action in config-browser/."

var nvdValue107 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue106}

var nvdValue108 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue107}, ID: "CVE-2013-6348", LastModified: "2026-06-17T00:00:20.440", Metrics: &nvdValue10, Published: "2013-11-02T21:55:04.630"}

var nvdValue109 = "The ParametersInterceptor in Apache Struts before 2.3.16.2 allows remote attackers to \"manipulate\" the ClassLoader via the class parameter, which is passed to the getClass method."

var nvdValue110 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue109}

var nvdValue111 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue110}, ID: "CVE-2014-0094", LastModified: "2026-06-17T00:02:14.987", Metrics: &nvdValue16, Published: "2014-03-11T13:00:37.107"}

var nvdValue112 = "ParametersInterceptor in Apache Struts before 2.3.20 does not properly restrict access to the getClass method, which allows remote attackers to \"manipulate\" the ClassLoader and execute arbitrary code via a crafted request. NOTE: this vulnerability exists because of an incomplete fix for CVE-2014-0094."

var nvdValue113 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue112}

var nvdValue114 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue27}}

var nvdValue115 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue113}, ID: "CVE-2014-0112", LastModified: "2026-06-17T00:02:17.950", Metrics: &nvdValue114, Published: "2014-04-29T10:37:03.670"}

var nvdValue116 = "CookieInterceptor in Apache Struts before 2.3.20, when a wildcard cookiesName value is used, does not properly restrict access to the getClass method, which allows remote attackers to \"manipulate\" the ClassLoader and execute arbitrary code via a crafted request. NOTE: this vulnerability exists because of an incomplete fix for CVE-2014-0094."

var nvdValue117 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue116}

var nvdValue118 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue117}, ID: "CVE-2014-0113", LastModified: "2026-06-17T00:02:18.107", Metrics: &nvdValue114, Published: "2014-04-29T10:37:03.700"}

var nvdValue119 = "CookieInterceptor in Apache Struts 2.x before 2.3.20, when a wildcard cookiesName value is used, does not properly restrict access to the getClass method, which allows remote attackers to \"manipulate\" the ClassLoader and modify session state via a crafted request. NOTE: this vulnerability exists because of an incomplete fix for CVE-2014-0113."

var nvdValue120 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue119}

var nvdValue121 = schema.CVSSV20{BaseScore: 5.8, VectorString: "AV:N/AC:M/Au:N/C:N/I:P/A:P", Version: "2.0"}

var nvdValue122 = schema.CVEAPIJSON20CVSSV2{CvssData: &nvdValue121}

var nvdValue123 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue122}}

var nvdValue124 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue120}, ID: "CVE-2014-0116", LastModified: "2026-06-17T00:02:18.910", Metrics: &nvdValue123, Published: "2014-05-08T10:55:02.967"}

var nvdValue125 = "Apache Struts 2.0.0 through 2.3.x before 2.3.20 uses predictable <s:token/> values, which allows remote attackers to bypass the CSRF protection mechanism."

var nvdValue126 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue125}

var nvdValue127 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue126}, ID: "CVE-2014-7809", LastModified: "2026-06-17T00:15:43.807", Metrics: &nvdValue48, Published: "2014-12-10T15:59:01.347"}

var nvdValue128 = "The default exclude patterns (excludeParams) in Apache Struts 2.3.20 allow remote attackers to \"compromise internal state of an application\" via unspecified vectors."

var nvdValue129 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue128}

var nvdValue130 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue129}, ID: "CVE-2015-1831", LastModified: "2026-06-17T00:23:04.813", Metrics: &nvdValue114, Published: "2015-07-16T14:59:00.073"}

var nvdValue131 = "Apache Struts 2.x before 2.3.24.1 allows remote attackers to manipulate Struts internals, alter user sessions, or affect container settings via vectors involving a top object."

var nvdValue132 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue131}

var nvdValue133 = schema.CVSSV30{BaseScore: 7.5, VectorString: "CVSS:3.0/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:H/A:N", Version: "3.0"}

var nvdValue134 = schema.CVEAPIJSON20CVSSV30{CvssData: &nvdValue133}

var nvdValue135 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue15}, CvssMetricV30: []*schema.CVEAPIJSON20CVSSV30{&nvdValue134}}

var nvdValue136 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue132}, ID: "CVE-2015-5209", LastModified: "2026-06-17T00:28:40.667", Metrics: &nvdValue135, Published: "2017-08-29T15:29:00.393"}

var nvdValue137 = schema.CVEAPIJSON20LangString{Lang: "en", Value: "Cross-site scripting (XSS) vulnerability in Apache Struts before 2.3.20."}

var nvdValue138 = schema.CVSSV30{BaseScore: 6.1, VectorString: "CVSS:3.0/AV:N/AC:L/PR:N/UI:R/S:C/C:L/I:L/A:N", Version: "3.0"}

var nvdValue139 = schema.CVEAPIJSON20CVSSV30{CvssData: &nvdValue138}

var nvdValue140 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue9}, CvssMetricV30: []*schema.CVEAPIJSON20CVSSV30{&nvdValue139}}

var nvdValue141 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue137}, ID: "CVE-2015-5169", LastModified: "2026-06-17T00:28:36.103", Metrics: &nvdValue140, Published: "2017-09-25T21:29:00.303"}

var nvdValue142 = schema.CVEAPIJSON20LangString{Lang: "en", Value: "Apache Struts before 2.3.20 has a cross-site scripting (XSS) vulnerability."}

var nvdValue143 = schema.CVSSV31{BaseScore: 6.1, VectorString: "CVSS:3.1/AV:N/AC:L/PR:N/UI:R/S:C/C:L/I:L/A:N", Version: "3.1"}

var nvdValue144 = schema.CVEAPIJSON20CVSSV31{CvssData: &nvdValue143}

var nvdValue145 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue9}, CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue144}}

var nvdValue146 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue142}, ID: "CVE-2015-2992", LastModified: "2026-06-17T00:25:06.620", Metrics: &nvdValue145, Published: "2020-02-27T18:15:11.123"}

var nvdValue147 = "Apache Struts 2.x before 2.3.28 allows remote attackers to execute arbitrary code via a \"%{}\" sequence in a tag attribute, aka forced double OGNL evaluation."

var nvdValue148 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue147}

var nvdValue149 = schema.CVSSV20{BaseScore: 9, VectorString: "AV:N/AC:L/Au:S/C:C/I:C/A:C", Version: "2.0"}

var nvdValue150 = schema.CVEAPIJSON20CVSSV2{CvssData: &nvdValue149}

var nvdValue151 = schema.CVSSV30{BaseScore: 8.8, VectorString: "CVSS:3.0/AV:N/AC:L/PR:L/UI:N/S:U/C:H/I:H/A:H", Version: "3.0"}

var nvdValue152 = schema.CVEAPIJSON20CVSSV30{CvssData: &nvdValue151}

var nvdValue153 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue150}, CvssMetricV30: []*schema.CVEAPIJSON20CVSSV30{&nvdValue152}}

var nvdValue154 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue148}, ID: "CVE-2016-0785", LastModified: "2026-06-17T00:38:13.673", Metrics: &nvdValue153, Published: "2016-04-12T16:59:00.123"}

var nvdValue155 = "Apache Struts 2.x before 2.3.25 does not sanitize text in the Locale object constructed by I18NInterceptor, which might allow remote attackers to conduct cross-site scripting (XSS) attacks via unspecified vectors involving language display."

var nvdValue156 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue155}

var nvdValue157 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue156}, ID: "CVE-2016-2162", LastModified: "2026-06-17T00:43:33.760", Metrics: &nvdValue140, Published: "2016-04-12T16:59:01.203"}

var nvdValue158 = "Cross-site scripting (XSS) vulnerability in the URLDecoder function in JRE before 1.8, as used in Apache Struts 2.x before 2.3.28, when using a single byte page encoding, allows remote attackers to inject arbitrary web script or HTML via multi-byte characters in a url-encoded parameter."

var nvdValue159 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue158}

var nvdValue160 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue159}, ID: "CVE-2016-4003", LastModified: "2026-06-17T00:46:43.277", Metrics: &nvdValue140, Published: "2016-04-12T16:59:04.313"}

var nvdValue161 = "Apache Struts 2.3.19 to 2.3.20.2, 2.3.21 to 2.3.24.1, and 2.3.25 to 2.3.28, when Dynamic Method Invocation is enabled, allow remote attackers to execute arbitrary code via method: prefix, related to chained expressions."

var nvdValue162 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue161}

var nvdValue163 = schema.CVSSV30{BaseScore: 8.1, VectorString: "CVSS:3.0/AV:N/AC:H/PR:N/UI:N/S:U/C:H/I:H/A:H", Version: "3.0"}

var nvdValue164 = schema.CVEAPIJSON20CVSSV30{CvssData: &nvdValue163}

var nvdValue165 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue41}, CvssMetricV30: []*schema.CVEAPIJSON20CVSSV30{&nvdValue164}}

var nvdValue166 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue162}, ID: "CVE-2016-3081", LastModified: "2026-06-17T00:44:55.973", Metrics: &nvdValue165, Published: "2016-04-26T14:59:02.207"}

var nvdValue167 = "XSLTResult in Apache Struts 2.x before 2.3.20.2, 2.3.24.x before 2.3.24.2, and 2.3.28.x before 2.3.28.1 allows remote attackers to execute arbitrary code via the stylesheet location parameter."

var nvdValue168 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue167}

var nvdValue169 = schema.CVSSV30{BaseScore: 9.8, VectorString: "CVSS:3.0/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H", Version: "3.0"}

var nvdValue170 = schema.CVEAPIJSON20CVSSV30{CvssData: &nvdValue169}

var nvdValue171 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue59}, CvssMetricV30: []*schema.CVEAPIJSON20CVSSV30{&nvdValue170}}

var nvdValue172 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue168}, ID: "CVE-2016-3082", LastModified: "2026-06-17T00:44:56.117", Metrics: &nvdValue171, Published: "2016-04-26T14:59:03.190"}

var nvdValue173 = "Apache Struts 2.3.19 to 2.3.20.2, 2.3.21 to 2.3.24.1, and 2.3.25 to 2.3.28, when Dynamic Method Invocation is enabled, allow remote attackers to execute arbitrary code via vectors related to an ! (exclamation mark) operator to the REST Plugin."

var nvdValue174 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue173}

var nvdValue175 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue27}, CvssMetricV30: []*schema.CVEAPIJSON20CVSSV30{&nvdValue170}}

var nvdValue176 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue174}, ID: "CVE-2016-3087", LastModified: "2026-06-17T00:44:56.647", Metrics: &nvdValue175, Published: "2016-06-07T18:59:02.713"}

var nvdValue177 = "Apache Struts 2.0.0 through 2.3.24.1 does not properly cache method references when used with OGNL before 3.0.12, which allows remote attackers to cause a denial of service (block access to a web site) via unspecified vectors."

var nvdValue178 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue177}

var nvdValue179 = schema.CVSSV20{BaseScore: 5, VectorString: "AV:N/AC:L/Au:N/C:N/I:N/A:P", Version: "2.0"}

var nvdValue180 = schema.CVEAPIJSON20CVSSV2{CvssData: &nvdValue179}

var nvdValue181 = schema.CVSSV30{BaseScore: 5.3, VectorString: "CVSS:3.0/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:L", Version: "3.0"}

var nvdValue182 = schema.CVEAPIJSON20CVSSV30{CvssData: &nvdValue181}

var nvdValue183 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue180}, CvssMetricV30: []*schema.CVEAPIJSON20CVSSV30{&nvdValue182}}

var nvdValue184 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue178}, ID: "CVE-2016-3093", LastModified: "2026-06-17T00:44:57.577", Metrics: &nvdValue183, Published: "2016-06-07T18:59:03.683"}

var nvdValue185 = "The REST plugin in Apache Struts 2 2.3.19 through 2.3.28.1 allows remote attackers to execute arbitrary code via a crafted expression."

var nvdValue186 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue185}

var nvdValue187 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue186}, ID: "CVE-2016-4438", LastModified: "2026-06-17T00:47:33.507", Metrics: &nvdValue175, Published: "2016-07-04T22:59:09.100"}

var nvdValue188 = "The URLValidator class in Apache Struts 2 2.3.20 through 2.3.28.1 and 2.5.x before 2.5.1 allows remote attackers to cause a denial of service via a null value for a URL field."

var nvdValue189 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue188}

var nvdValue190 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue189}, ID: "CVE-2016-4465", LastModified: "2026-06-17T00:47:36.567", Metrics: &nvdValue183, Published: "2016-07-04T22:59:10.117"}

var nvdValue191 = "Use-after-free vulnerability in libxml2 through 2.9.4, as used in Google Chrome before 52.0.2743.82, allows remote attackers to cause a denial of service or possibly have unspecified other impact via vectors related to the XPointer range-to function."

var nvdValue192 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue191}

var nvdValue193 = schema.CVSSV30{BaseScore: 8.8, VectorString: "CVSS:3.0/AV:N/AC:L/PR:N/UI:R/S:U/C:H/I:H/A:H", Version: "3.0"}

var nvdValue194 = schema.CVEAPIJSON20CVSSV30{CvssData: &nvdValue193}

var nvdValue195 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue47}, CvssMetricV30: []*schema.CVEAPIJSON20CVSSV30{&nvdValue194}}

var nvdValue196 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue192}, ID: "CVE-2016-5131", LastModified: "2026-06-17T00:48:48.527", Metrics: &nvdValue195, Published: "2016-07-23T19:59:13.767"}

var nvdValue197 = "Apache Struts 2 before 2.3.29 and 2.5.x before 2.5.1 allow attackers to have unspecified impact via vectors related to improper action name clean up."

var nvdValue198 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue197}

var nvdValue199 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue198}, ID: "CVE-2016-4436", LastModified: "2026-06-17T00:47:33.200", Metrics: &nvdValue175, Published: "2016-10-03T15:59:01.913"}

var nvdValue200 = "libxml2 2.9.4 and earlier, as used in XMLSec 1.2.23 and earlier and other products, does not offer a flag directly indicating that the current document may be read but other files may not be opened, which makes it easier for remote attackers to conduct XML External Entity (XXE) attacks via a crafted document."

var nvdValue201 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue200}

var nvdValue202 = schema.CVSSV20{BaseScore: 4.3, VectorString: "AV:N/AC:M/Au:N/C:P/I:N/A:N", Version: "2.0"}

var nvdValue203 = schema.CVEAPIJSON20CVSSV2{CvssData: &nvdValue202}

var nvdValue204 = schema.CVSSV31{BaseScore: 5.5, VectorString: "CVSS:3.1/AV:L/AC:L/PR:N/UI:R/S:U/C:H/I:N/A:N", Version: "3.1"}

var nvdValue205 = schema.CVEAPIJSON20CVSSV31{CvssData: &nvdValue204}

var nvdValue206 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue203}, CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue205}}

var nvdValue207 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue201}, ID: "CVE-2016-9318", LastModified: "2026-06-17T00:55:49.887", Metrics: &nvdValue206, Published: "2016-11-16T00:59:00.180"}

var nvdValue208 = "Multiple integer overflows in the TRE library and musl libc allow attackers to cause memory corruption via a large number of (1) states or (2) tags, which triggers an out-of-bounds write."

var nvdValue209 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue208}

var nvdValue210 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue209}, ID: "CVE-2016-8859", LastModified: "2026-06-17T00:55:05.520", Metrics: &nvdValue175, Published: "2017-02-13T18:59:00.753"}

var nvdValue211 = "In Apache Struts 2.5 through 2.5.5, if an application allows entering a URL in a form field and the built-in URLValidator is used, it is possible to prepare a special URL which will be used to overload server process when performing validation of the URL."

var nvdValue212 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue211}

var nvdValue213 = schema.CVSSV20{BaseScore: 4.3, VectorString: "AV:N/AC:M/Au:N/C:N/I:N/A:P", Version: "2.0"}

var nvdValue214 = schema.CVEAPIJSON20CVSSV2{CvssData: &nvdValue213}

var nvdValue215 = schema.CVSSV30{BaseScore: 5.9, VectorString: "CVSS:3.0/AV:N/AC:H/PR:N/UI:N/S:U/C:N/I:N/A:H", Version: "3.0"}

var nvdValue216 = schema.CVEAPIJSON20CVSSV30{CvssData: &nvdValue215}

var nvdValue217 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue214}, CvssMetricV30: []*schema.CVEAPIJSON20CVSSV30{&nvdValue216}}

var nvdValue218 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue212}, ID: "CVE-2016-8738", LastModified: "2026-06-17T00:54:55.010", Metrics: &nvdValue217, Published: "2017-09-20T17:29:00.337"}

var nvdValue219 = "Apache Struts 2.x before 2.3.29 allows remote attackers to execute arbitrary code via a \"%{}\" sequence in a tag attribute, aka forced double OGNL evaluation.  NOTE: this vulnerability exists because of an incomplete fix for CVE-2016-0785."

var nvdValue220 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue219}

var nvdValue221 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue220}, ID: "CVE-2016-4461", LastModified: "2026-06-17T00:47:36.053", Metrics: &nvdValue153, Published: "2017-10-16T16:29:00.607"}

var nvdValue222 = "The Jakarta Multipart parser in Apache Struts 2 2.3.x before 2.3.32 and 2.5.x before 2.5.10.1 has incorrect exception handling and error-message generation during file-upload attempts, which allows remote attackers to execute arbitrary commands via a crafted Content-Type, Content-Disposition, or Content-Length HTTP header, as exploited in the wild in March 2017 with a Content-Type header containing a #cmd= string."

var nvdValue223 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue222}

var nvdValue224 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue59}, CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue29}}

var nvdValue225 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue223}, ID: "CVE-2017-5638", LastModified: "2026-06-17T01:20:54.013", Metrics: &nvdValue224, Published: "2017-03-11T02:59:00.150"}

var nvdValue226 = "libpcre1 in PCRE 8.40 and libpcre2 in PCRE2 10.23 allow remote attackers to cause a denial of service (segmentation violation for read access, and application crash) by triggering an invalid Unicode property lookup."

var nvdValue227 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue226}

var nvdValue228 = schema.CVSSV30{BaseScore: 7.5, VectorString: "CVSS:3.0/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", Version: "3.0"}

var nvdValue229 = schema.CVEAPIJSON20CVSSV30{CvssData: &nvdValue228}

var nvdValue230 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue180}, CvssMetricV30: []*schema.CVEAPIJSON20CVSSV30{&nvdValue229}}

var nvdValue231 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue227}, ID: "CVE-2017-7186", LastModified: "2026-06-17T01:23:49.857", Metrics: &nvdValue230, Published: "2017-03-20T00:59:00.190"}

var nvdValue232 = "libxml2 2.9.4, when used in recover mode, allows remote attackers to cause a denial of service (NULL pointer dereference) via a crafted XML document.  NOTE: The maintainer states \"I would disagree of a CVE with the Recover parsing option which should only be used for manual recovery at least for XML parser."

var nvdValue233 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue232}

var nvdValue234 = schema.CVSSV20{BaseScore: 2.6, VectorString: "AV:N/AC:H/Au:N/C:N/I:N/A:P", Version: "2.0"}

var nvdValue235 = schema.CVEAPIJSON20CVSSV2{CvssData: &nvdValue234}

var nvdValue236 = schema.CVSSV30{BaseScore: 4.7, VectorString: "CVSS:3.0/AV:L/AC:H/PR:N/UI:R/S:U/C:N/I:N/A:H", Version: "3.0"}

var nvdValue237 = schema.CVEAPIJSON20CVSSV30{CvssData: &nvdValue236}

var nvdValue238 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue235}, CvssMetricV30: []*schema.CVEAPIJSON20CVSSV30{&nvdValue237}}

var nvdValue239 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue233}, ID: "CVE-2017-5969", LastModified: "2026-06-17T01:21:31.863", Metrics: &nvdValue238, Published: "2017-04-11T16:59:00.343"}

var nvdValue240 = "FreeType 2 before 2017-03-24 has an out-of-bounds write caused by a heap-based buffer overflow related to the t1_decoder_parse_charstrings function in psaux/t1decode.c."

var nvdValue241 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue240}

var nvdValue242 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue241}, ID: "CVE-2017-8105", LastModified: "2026-06-17T01:25:46.873", Metrics: &nvdValue175, Published: "2017-04-24T18:59:00.897"}

var nvdValue243 = "The xsltAddTextString function in transform.c in libxslt 1.1.29, as used in Blink in Google Chrome prior to 57.0.2987.98 for Mac, Windows, and Linux and 57.0.2987.108 for Android, lacked a check for integer overflow during a size calculation, which allowed a remote attacker to perform an out of bounds memory write via a crafted HTML page."

var nvdValue244 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue243}

var nvdValue245 = schema.CVSSV31{BaseScore: 8.8, VectorString: "CVSS:3.1/AV:N/AC:L/PR:N/UI:R/S:U/C:H/I:H/A:H", Version: "3.1"}

var nvdValue246 = schema.CVEAPIJSON20CVSSV31{CvssData: &nvdValue245}

var nvdValue247 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue47}, CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue246}}

var nvdValue248 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue244}, ID: "CVE-2017-5029", LastModified: "2026-06-17T01:19:46.610", Metrics: &nvdValue247, Published: "2017-04-24T23:59:00.157"}

var nvdValue249 = "FreeType 2 before 2017-03-26 has an out-of-bounds write caused by a heap-based buffer overflow related to the t1_builder_close_contour function in psaux/psobjs.c."

var nvdValue250 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue249}

var nvdValue251 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue250}, ID: "CVE-2017-8287", LastModified: "2026-06-17T01:26:07.850", Metrics: &nvdValue175, Published: "2017-04-27T00:59:00.320"}

var nvdValue252 = "In the cron package through 3.0pl1-128 on Debian, and through 3.0pl1-128ubuntu2 on Ubuntu, the postinst maintainer script allows for group-crontab-to-root privilege escalation via symlink attacks against unsafe usage of the chown and chmod programs."

var nvdValue253 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue252}

var nvdValue254 = schema.CVSSV20{BaseScore: 6.9, VectorString: "AV:L/AC:M/Au:N/C:C/I:C/A:C", Version: "2.0"}

var nvdValue255 = schema.CVEAPIJSON20CVSSV2{CvssData: &nvdValue254}

var nvdValue256 = schema.CVSSV31{BaseScore: 6.7, VectorString: "CVSS:3.1/AV:L/AC:L/PR:H/UI:N/S:U/C:H/I:H/A:H", Version: "3.1"}

var nvdValue257 = schema.CVEAPIJSON20CVSSV31{CvssData: &nvdValue256}

var nvdValue258 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue255}, CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue257}}

var nvdValue259 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue253}, ID: "CVE-2017-9525", LastModified: "2026-06-17T01:28:18.113", Metrics: &nvdValue258, Published: "2017-06-09T16:29:02.110"}

var nvdValue260 = "In PCRE 8.41, the OP_KETRMAX feature in the match function in pcre_exec.c allows stack exhaustion (uncontrolled recursion) when processing a crafted regular expression."

var nvdValue261 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue260}

var nvdValue262 = schema.CVSSV20{BaseScore: 7.8, VectorString: "AV:N/AC:L/Au:N/C:N/I:N/A:C", Version: "2.0"}

var nvdValue263 = schema.CVEAPIJSON20CVSSV2{CvssData: &nvdValue262}

var nvdValue264 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue263}, CvssMetricV30: []*schema.CVEAPIJSON20CVSSV30{&nvdValue229}}

var nvdValue265 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue261}, ID: "CVE-2017-11164", LastModified: "2026-06-17T01:01:20.163", Metrics: &nvdValue264, Published: "2017-07-11T03:29:00.277"}

var nvdValue266 = "Nginx versions since 0.5.6 up to and including 1.13.2 are vulnerable to integer overflow vulnerability in nginx range filter module resulting into leak of potentially sensitive information triggered by specially crafted request."

var nvdValue267 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue266}

var nvdValue268 = schema.CVSSV31{BaseScore: 7.5, VectorString: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:N/A:N", Version: "3.1"}

var nvdValue269 = schema.CVEAPIJSON20CVSSV31{CvssData: &nvdValue268}

var nvdValue270 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue3}, CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue269}}

var nvdValue271 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue267}, ID: "CVE-2017-7529", LastModified: "2026-06-17T01:24:32.470", Metrics: &nvdValue270, Published: "2017-07-13T13:29:00.220"}

var nvdValue272 = "If an application allows enter an URL in a form field and built-in URLValidator is used, it is possible to prepare a special URL which will be used to overload server process when performing validation of the URL. Solution is to upgrade to Apache Struts version 2.5.12."

var nvdValue273 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue272}

var nvdValue274 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue273}, ID: "CVE-2017-7672", LastModified: "2026-06-17T01:24:56.840", Metrics: &nvdValue217, Published: "2017-07-13T15:29:00.363"}

var nvdValue275 = "When using a Spring AOP functionality to secure Struts actions it is possible to perform a DoS attack. Solution is to upgrade to Apache Struts version 2.5.12 or 2.3.33."

var nvdValue276 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue275}

var nvdValue277 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue276}, ID: "CVE-2017-9787", LastModified: "2026-06-17T01:28:54.133", Metrics: &nvdValue230, Published: "2017-07-13T15:29:00.393"}

var nvdValue278 = "The fill_input_buffer function in jdatasrc.c in libjpeg-turbo 1.5.1 allows remote attackers to cause a denial of service (invalid memory access and application crash) or possibly have unspecified other impact via a crafted jpg file. NOTE: Maintainer asserts the issue is due to a bug in downstream code caused by misuse of the libjpeg API"

var nvdValue279 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue278}

var nvdValue280 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue279}, ID: "CVE-2017-9614", LastModified: "2026-06-17T01:28:34.913", Metrics: &nvdValue247, Published: "2017-07-27T06:29:00.897"}

var nvdValue281 = "In Apache Struts 2.0.0 through 2.3.33 and 2.5 through 2.5.10.1, using an unintentional expression in a Freemarker tag instead of string literals can lead to a RCE attack."

var nvdValue282 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue281}

var nvdValue283 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue282}, ID: "CVE-2017-12611", LastModified: "2026-06-17T01:03:37.133", Metrics: &nvdValue175, Published: "2017-09-20T17:29:00.400"}

var nvdValue284 = "In Apache Struts 2.3.7 through 2.3.33 and 2.5 through 2.5.12, if an application allows entering a URL in a form field and built-in URLValidator is used, it is possible to prepare a special URL which will be used to overload server process when performing validation of the URL.  NOTE: this vulnerability exists because of an incomplete fix for S2-047 / CVE-2017-7672."

var nvdValue285 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue284}

var nvdValue286 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue285}, ID: "CVE-2017-9804", LastModified: "2026-06-17T01:28:57.343", Metrics: &nvdValue230, Published: "2017-09-20T17:29:00.620"}

var nvdValue287 = "The get_next_block function in archival/libarchive/decompress_bunzip2.c in BusyBox 1.27.2 has an Integer Overflow that may lead to a write access violation."

var nvdValue288 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue287}

var nvdValue289 = schema.CVSSV31{BaseScore: 5.5, VectorString: "CVSS:3.1/AV:L/AC:L/PR:N/UI:R/S:U/C:N/I:N/A:H", Version: "3.1"}

var nvdValue290 = schema.CVEAPIJSON20CVSSV31{CvssData: &nvdValue289}

var nvdValue291 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue214}, CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue290}}

var nvdValue292 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue288}, ID: "CVE-2017-15873", LastModified: "2026-06-17T01:08:24.563", Metrics: &nvdValue291, Published: "2017-10-24T20:29:00.280"}

var nvdValue293 = "archival/libarchive/decompress_unlzma.c in BusyBox 1.27.2 has an Integer Underflow that leads to a read access violation."

var nvdValue294 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue293}

var nvdValue295 = schema.CVSSV30{BaseScore: 5.5, VectorString: "CVSS:3.0/AV:L/AC:L/PR:N/UI:R/S:U/C:N/I:N/A:H", Version: "3.0"}

var nvdValue296 = schema.CVEAPIJSON20CVSSV30{CvssData: &nvdValue295}

var nvdValue297 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue214}, CvssMetricV30: []*schema.CVEAPIJSON20CVSSV30{&nvdValue296}}

var nvdValue298 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue294}, ID: "CVE-2017-15874", LastModified: "2026-06-17T01:08:24.750", Metrics: &nvdValue297, Published: "2017-10-24T20:29:00.327"}

var nvdValue299 = "VIM version 8.0.1187 (and other versions most likely) ignores umask when creating a swap file (\"[ORIGINAL_FILENAME].swp\") resulting in files that may be world readable or otherwise accessible in ways not intended by the user running the vi binary."

var nvdValue300 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue299}

var nvdValue301 = schema.CVSSV20{BaseScore: 2.1, VectorString: "AV:L/AC:L/Au:N/C:P/I:N/A:N", Version: "2.0"}

var nvdValue302 = schema.CVEAPIJSON20CVSSV2{CvssData: &nvdValue301}

var nvdValue303 = schema.CVSSV30{BaseScore: 5.5, VectorString: "CVSS:3.0/AV:L/AC:L/PR:L/UI:N/S:U/C:H/I:N/A:N", Version: "3.0"}

var nvdValue304 = schema.CVEAPIJSON20CVSSV30{CvssData: &nvdValue303}

var nvdValue305 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue302}, CvssMetricV30: []*schema.CVEAPIJSON20CVSSV30{&nvdValue304}}

var nvdValue306 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue300}, ID: "CVE-2017-1000382", LastModified: "2026-06-17T00:59:04.350", Metrics: &nvdValue305, Published: "2017-10-31T20:29:00.263"}

var nvdValue307 = "In the add_match function in libbb/lineedit.c in BusyBox through 1.27.2, the tab autocomplete feature of the shell, used to get a list of filenames in a directory, does not sanitize filenames and results in executing any escape sequence in the terminal. This could potentially result in code execution, arbitrary file writes, or other attacks."

var nvdValue308 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue307}

var nvdValue309 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue308}, ID: "CVE-2017-16544", LastModified: "2026-06-17T01:09:26.423", Metrics: &nvdValue71, Published: "2017-11-20T15:29:00.387"}

var nvdValue310 = "In PCRE 8.41, after compiling, a pcretest load test PoC produces a crash overflow in the function match() in pcre_exec.c because of a self-recursive call. NOTE: third parties dispute the relevance of this report, noting that there are options that can be used to limit the amount of stack that is used"

var nvdValue311 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue310}

var nvdValue312 = schema.CVSSV20{BaseScore: 2.1, VectorString: "AV:L/AC:L/Au:N/C:N/I:N/A:P", Version: "2.0"}

var nvdValue313 = schema.CVEAPIJSON20CVSSV2{CvssData: &nvdValue312}

var nvdValue314 = schema.CVSSV30{BaseScore: 5.5, VectorString: "CVSS:3.0/AV:L/AC:L/PR:L/UI:N/S:U/C:N/I:N/A:H", Version: "3.0"}

var nvdValue315 = schema.CVEAPIJSON20CVSSV30{CvssData: &nvdValue314}

var nvdValue316 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue313}, CvssMetricV30: []*schema.CVEAPIJSON20CVSSV30{&nvdValue315}}

var nvdValue317 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue311}, ID: "CVE-2017-16231", LastModified: "2026-06-17T01:09:01.150", Metrics: &nvdValue316, Published: "2019-03-21T15:59:56.217"}

var nvdValue318 = "An issue was discovered in FreeType 2 through 2.9. A NULL pointer dereference in the Ins_GETVARIATION() function within ttinterp.c could lead to DoS via a crafted font file."

var nvdValue319 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue318}

var nvdValue320 = schema.CVSSV30{BaseScore: 6.5, VectorString: "CVSS:3.0/AV:N/AC:L/PR:N/UI:R/S:U/C:N/I:N/A:H", Version: "3.0"}

var nvdValue321 = schema.CVEAPIJSON20CVSSV30{CvssData: &nvdValue320}

var nvdValue322 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue214}, CvssMetricV30: []*schema.CVEAPIJSON20CVSSV30{&nvdValue321}}

var nvdValue323 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue319}, ID: "CVE-2018-6942", LastModified: "2026-06-17T02:02:29.537", Metrics: &nvdValue322, Published: "2018-02-13T05:29:00.267"}

var nvdValue324 = "The xz_decomp function in xzlib.c in libxml2 2.9.8, if --with-lzma is used, allows remote attackers to cause a denial of service (infinite loop) via a crafted XML file that triggers LZMA_MEMLIMIT_ERROR, as demonstrated by xmllint, a different vulnerability than CVE-2015-8035."

var nvdValue325 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue324}

var nvdValue326 = schema.CVSSV30{BaseScore: 5.3, VectorString: "CVSS:3.0/AV:N/AC:H/PR:N/UI:R/S:U/C:N/I:N/A:H", Version: "3.0"}

var nvdValue327 = schema.CVEAPIJSON20CVSSV30{CvssData: &nvdValue326}

var nvdValue328 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue235}, CvssMetricV30: []*schema.CVEAPIJSON20CVSSV30{&nvdValue327}}

var nvdValue329 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue325}, ID: "CVE-2018-9251", LastModified: "2026-06-17T02:06:17.680", Metrics: &nvdValue328, Published: "2018-04-04T02:29:00.320"}

var nvdValue330 = "The Node.js inspector, in 6.x and later is vulnerable to a DNS rebinding attack which could be exploited to perform remote code execution. An attack is possible from malicious websites open in a web browser on the same computer, or another computer with network access to the computer running the Node.js process. A malicious website could use a DNS rebinding attack to trick the web browser to bypass same-origin-policy checks and to allow HTTP connections to localhost or to hosts on the local network. If a Node.js process with the debug port active is running on localhost or on a host on the local network, the malicious website could connect to it as a debugger, and get full code execution access."

var nvdValue331 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue330}

var nvdValue332 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue331}, ID: "CVE-2018-7160", LastModified: "2026-06-17T02:02:43.293", Metrics: &nvdValue247, Published: "2018-05-17T14:29:00.827"}

var nvdValue333 = schema.CVEAPIJSON20LangString{Lang: "en", Value: "libjpeg 9c has a large loop because read_pixel in rdtarga.c mishandles EOF."}

var nvdValue334 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue333}, ID: "CVE-2018-11813", LastModified: "2026-06-17T01:36:38.723", Metrics: &nvdValue230, Published: "2018-06-06T03:29:00.297"}

var nvdValue335 = "Libgcrypt before 1.7.10 and 1.8.x before 1.8.3 allows a memory-cache side-channel attack on ECDSA signatures that can be mitigated through the use of blinding during the signing process in the _gcry_ecc_ecdsa_sign function in cipher/ecc-ecdsa.c, aka the Return Of the Hidden Number Problem or ROHNP. To discover an ECDSA key, the attacker needs access to either the local machine or a different virtual machine on the same physical host."

var nvdValue336 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue335}

var nvdValue337 = schema.CVSSV20{BaseScore: 1.9, VectorString: "AV:L/AC:M/Au:N/C:P/I:N/A:N", Version: "2.0"}

var nvdValue338 = schema.CVEAPIJSON20CVSSV2{CvssData: &nvdValue337}

var nvdValue339 = schema.CVSSV30{BaseScore: 4.7, VectorString: "CVSS:3.0/AV:L/AC:H/PR:L/UI:N/S:U/C:H/I:N/A:N", Version: "3.0"}

var nvdValue340 = schema.CVEAPIJSON20CVSSV30{CvssData: &nvdValue339}

var nvdValue341 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue338}, CvssMetricV30: []*schema.CVEAPIJSON20CVSSV30{&nvdValue340}}

var nvdValue342 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue336}, ID: "CVE-2018-0495", LastModified: "2026-06-17T01:30:57.543", Metrics: &nvdValue341, Published: "2018-06-13T23:29:00.333"}

var nvdValue343 = "libjpeg-turbo 1.5.90 is vulnerable to a denial of service vulnerability caused by a divide by zero when processing a crafted BMP image."

var nvdValue344 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue343}

var nvdValue345 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue344}, ID: "CVE-2018-1152", LastModified: "2026-06-17T01:50:35.720", Metrics: &nvdValue322, Published: "2018-06-18T14:29:00.323"}

var nvdValue346 = "Busybox contains a Missing SSL certificate validation vulnerability in The \"busybox wget\" applet that can result in arbitrary code execution. This attack appear to be exploitable via Simply download any file over HTTPS using \"busybox wget https://compromised-domain.com/important-file\"."

var nvdValue347 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue346}

var nvdValue348 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue47}, CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue83}}

var nvdValue349 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue347}, ID: "CVE-2018-1000500", LastModified: "2026-06-17T01:32:47.100", Metrics: &nvdValue348, Published: "2018-06-26T16:29:00.353"}

var nvdValue350 = "An issue has been found in libpng 1.6.34. It is a SEGV in the function png_free_data in png.c, related to the recommended error handling for png_read_image."

var nvdValue351 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue350}

var nvdValue352 = schema.CVSSV31{BaseScore: 6.5, VectorString: "CVSS:3.1/AV:N/AC:L/PR:N/UI:R/S:U/C:N/I:N/A:H", Version: "3.1"}

var nvdValue353 = schema.CVEAPIJSON20CVSSV31{CvssData: &nvdValue352}

var nvdValue354 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue214}, CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue353}}

var nvdValue355 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue351}, ID: "CVE-2018-14048", LastModified: "2026-06-17T01:40:32.630", Metrics: &nvdValue354, Published: "2018-07-13T16:29:00.377"}

var nvdValue356 = "A NULL pointer dereference vulnerability exists in the xpath.c:xmlXPathCompOpEval() function of libxml2 through 2.9.8 when parsing an invalid XPath expression in the XPATH_OP_AND or XPATH_OP_OR case. Applications processing untrusted XSL format inputs with the use of the libxml2 library may be vulnerable to a denial of service attack due to a crash of the application."

var nvdValue357 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue356}

var nvdValue358 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue357}, ID: "CVE-2018-14404", LastModified: "2026-06-17T01:40:57.037", Metrics: &nvdValue230, Published: "2018-07-19T13:29:00.480"}

var nvdValue359 = "libxml2 2.9.8, if --with-lzma is used, allows remote attackers to cause a denial of service (infinite loop) via a crafted XML file that triggers LZMA_MEMLIMIT_ERROR, as demonstrated by xmllint, a different vulnerability than CVE-2015-8035 and CVE-2018-9251."

var nvdValue360 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue359}

var nvdValue361 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue360}, ID: "CVE-2018-14567", LastModified: "2026-06-17T01:41:12.277", Metrics: &nvdValue322, Published: "2018-08-16T20:29:02.470"}

var nvdValue362 = "Apache Struts versions 2.3 to 2.3.34 and 2.5 to 2.5.16 suffer from possible Remote Code Execution when alwaysSelectFullNamespace is true (either by user or a plugin like Convention Plugin) and then: results are used with no namespace and in same time, its upper package have no or wildcard namespace and similar to results, same possibility when using url tag which doesn't have value and action set and in same time, its upper package have no or wildcard namespace."

var nvdValue363 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue362}

var nvdValue364 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue363}, ID: "CVE-2018-11776", LastModified: "2026-06-17T01:36:34.350", Metrics: &nvdValue84, Published: "2018-08-22T13:29:00.753"}

var nvdValue365 = "The OpenSSL ECDSA signature algorithm has been shown to be vulnerable to a timing side channel attack. An attacker could use variations in the signing algorithm to recover the private key. Fixed in OpenSSL 1.1.0j (Affected 1.1.0-1.1.0i). Fixed in OpenSSL 1.1.1a (Affected 1.1.1)."

var nvdValue366 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue365}

var nvdValue367 = schema.CVSSV31{BaseScore: 5.9, VectorString: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:H/I:N/A:N", Version: "3.1"}

var nvdValue368 = schema.CVEAPIJSON20CVSSV31{CvssData: &nvdValue367}

var nvdValue369 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue203}, CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue368}}

var nvdValue370 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue366}, ID: "CVE-2018-0735", LastModified: "2026-06-17T01:31:33.120", Metrics: &nvdValue369, Published: "2018-10-29T13:29:00.263"}

var nvdValue371 = "The OpenSSL DSA signature algorithm has been shown to be vulnerable to a timing side channel attack. An attacker could use variations in the signing algorithm to recover the private key. Fixed in OpenSSL 1.1.1a (Affected 1.1.1). Fixed in OpenSSL 1.1.0j (Affected 1.1.0-1.1.0i). Fixed in OpenSSL 1.0.2q (Affected 1.0.2-1.0.2p)."

var nvdValue372 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue371}

var nvdValue373 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue372}, ID: "CVE-2018-0734", LastModified: "2026-06-17T01:31:32.857", Metrics: &nvdValue369, Published: "2018-10-30T12:29:00.257"}

var nvdValue374 = "nginx before versions 1.15.6 and 1.14.1 has a vulnerability in the implementation of HTTP/2 that can allow for excessive memory consumption. This issue affects nginx compiled with the ngx_http_v2_module (not compiled by default) if the 'http2' option of the 'listen' directive is used in a configuration file."

var nvdValue375 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue374}

var nvdValue376 = schema.CVSSV31{BaseScore: 7.5, VectorString: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", Version: "3.1"}

var nvdValue377 = schema.CVEAPIJSON20CVSSV31{CvssData: &nvdValue376}

var nvdValue378 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue263}, CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue377}}

var nvdValue379 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue375}, ID: "CVE-2018-16843", LastModified: "2026-06-17T01:44:53.387", Metrics: &nvdValue378, Published: "2018-11-07T14:29:00.777"}

var nvdValue380 = "nginx before versions 1.15.6 and 1.14.1 has a vulnerability in the implementation of HTTP/2 that can allow for excessive CPU usage. This issue affects nginx compiled with the ngx_http_v2_module (not compiled by default) if the 'http2' option of the 'listen' directive is used in a configuration file."

var nvdValue381 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue380}

var nvdValue382 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue381}, ID: "CVE-2018-16844", LastModified: "2026-06-17T01:44:53.563", Metrics: &nvdValue378, Published: "2018-11-07T14:29:00.837"}

var nvdValue383 = "nginx before versions 1.15.6, 1.14.1 has a vulnerability in the ngx_http_mp4_module, which might allow an attacker to cause infinite loop in a worker process, cause a worker process crash, or might result in worker process memory disclosure by using a specially crafted mp4 file. The issue only affects nginx if it is built with the ngx_http_mp4_module (the module is not built by default) and the .mp4. directive is used in the configuration file. Further, the attack is only possible if an attacker is able to trigger processing of a specially crafted mp4 file with the ngx_http_mp4_module."

var nvdValue384 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue383}

var nvdValue385 = schema.CVSSV20{BaseScore: 5.8, VectorString: "AV:N/AC:M/Au:N/C:P/I:N/A:P", Version: "2.0"}

var nvdValue386 = schema.CVEAPIJSON20CVSSV2{CvssData: &nvdValue385}

var nvdValue387 = schema.CVSSV31{BaseScore: 6.1, VectorString: "CVSS:3.1/AV:L/AC:L/PR:N/UI:R/S:U/C:L/I:N/A:H", Version: "3.1"}

var nvdValue388 = schema.CVEAPIJSON20CVSSV31{CvssData: &nvdValue387}

var nvdValue389 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue386}, CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue388}}

var nvdValue390 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue384}, ID: "CVE-2018-16845", LastModified: "2026-06-17T01:44:53.720", Metrics: &nvdValue389, Published: "2018-11-07T14:29:00.883"}

var nvdValue391 = "Simultaneous Multi-threading (SMT) in processors can enable local users to exploit software vulnerable to timing attacks via a side-channel timing attack on 'port contention'."

var nvdValue392 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue391}

var nvdValue393 = schema.CVSSV31{BaseScore: 4.7, VectorString: "CVSS:3.1/AV:L/AC:H/PR:L/UI:N/S:U/C:H/I:N/A:N", Version: "3.1"}

var nvdValue394 = schema.CVEAPIJSON20CVSSV31{CvssData: &nvdValue393}

var nvdValue395 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue338}, CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue394}}

var nvdValue396 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue392}, ID: "CVE-2018-5407", LastModified: "2026-06-17T02:00:16.487", Metrics: &nvdValue395, Published: "2018-11-15T21:29:00.233"}

var nvdValue397 = "An issue was discovered in BusyBox before 1.30.0. An out of bounds read in udhcp components (consumed by the DHCP server, client, and relay) allows a remote attacker to leak sensitive information from the stack by sending a crafted DHCP message. This is related to verification in udhcp_get_option() in networking/udhcp/common.c that 4-byte options are indeed 4 bytes."

var nvdValue398 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue397}

var nvdValue399 = schema.CVSSV30{BaseScore: 7.5, VectorString: "CVSS:3.0/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:N/A:N", Version: "3.0"}

var nvdValue400 = schema.CVEAPIJSON20CVSSV30{CvssData: &nvdValue399}

var nvdValue401 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue3}, CvssMetricV30: []*schema.CVEAPIJSON20CVSSV30{&nvdValue400}}

var nvdValue402 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue398}, ID: "CVE-2018-20679", LastModified: "2026-06-17T01:53:18.477", Metrics: &nvdValue401, Published: "2019-01-09T16:29:00.273"}

var nvdValue403 = "get_8bit_row in rdbmp.c in libjpeg-turbo through 1.5.90 and MozJPEG through 3.3.1 allows attackers to cause a denial of service (heap-based buffer over-read and application crash) via a crafted 8-bit BMP in which one or more of the color indices is out of range for the number of palette entries."

var nvdValue404 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue403}

var nvdValue405 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue404}, ID: "CVE-2018-14498", LastModified: "2026-06-17T01:41:06.780", Metrics: &nvdValue322, Published: "2019-03-07T23:29:00.487"}

var nvdValue406 = "An issue has been found in third-party PNM decoding associated with libpng 1.6.35. It is a stack-based buffer overflow in the function get_token in pnm2png.c in pnm2png."

var nvdValue407 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue406}

var nvdValue408 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue407}, ID: "CVE-2018-14550", LastModified: "2026-06-17T01:41:10.890", Metrics: &nvdValue247, Published: "2019-07-10T12:15:10.750"}

var nvdValue409 = "An issue was discovered in BusyBox through 1.30.0. An out of bounds read in udhcp components (consumed by the DHCP client, server, and/or relay) might allow a remote attacker to leak sensitive information from the stack by sending a crafted DHCP message. This is related to assurance of a 4-byte length when decoding DHCP_SUBNET. NOTE: this issue exists because of an incomplete fix for CVE-2018-20679."

var nvdValue410 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue409}

var nvdValue411 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue410}, ID: "CVE-2019-5747", LastModified: "2026-06-17T02:38:08.073", Metrics: &nvdValue270, Published: "2019-01-09T16:29:00.353"}

var nvdValue412 = "png_image_free in png.c in libpng 1.6.x before 1.6.37 has a use-after-free because png_image_free_function is called under png_safe_execute."

var nvdValue413 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue412}

var nvdValue414 = schema.CVSSV31{BaseScore: 5.3, VectorString: "CVSS:3.1/AV:N/AC:H/PR:N/UI:R/S:U/C:N/I:N/A:H", Version: "3.1"}

var nvdValue415 = schema.CVEAPIJSON20CVSSV31{CvssData: &nvdValue414}

var nvdValue416 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue235}, CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue415}}

var nvdValue417 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue413}, ID: "CVE-2019-7317", LastModified: "2026-06-17T02:40:26.800", Metrics: &nvdValue416, Published: "2019-02-04T08:29:00.447"}

var nvdValue418 = "ChaCha20-Poly1305 is an AEAD cipher, and requires a unique nonce input for every encryption operation. RFC 7539 specifies that the nonce value (IV) should be 96 bits (12 bytes). OpenSSL allows a variable nonce length and front pads the nonce with 0 bytes if it is less than 12 bytes. However it also incorrectly allows a nonce to be set of up to 16 bytes. In this case only the last 12 bytes are significant and any additional leading bytes are ignored. It is a requirement of using this cipher that nonce values are unique. Messages encrypted using a reused nonce value are susceptible to serious confidentiality and integrity attacks. If an application changes the default nonce length to be longer than 12 bytes and then makes a change to the leading bytes of the nonce expecting the new value to be a new unique nonce then such an application could inadvertently encrypt messages with a reused nonce. Additionally the ignored bytes in a long nonce are not covered by the integrity guarantee of this cipher. Any application that relies on the integrity of these ignored leading bytes of a long nonce may be further affected. Any OpenSSL internal use of this cipher, including in SSL/TLS, is safe because no such use sets such a long nonce value. However user applications that use this cipher directly and set a non-default nonce length to be longer than 12 bytes may be vulnerable. OpenSSL versions 1.1.1 and 1.1.0 are affected by this issue. Due to the limited scope of affected deployments this has been assessed as low severity and therefore we are not creating new releases at this time. Fixed in OpenSSL 1.1.1c (Affected 1.1.1-1.1.1b). Fixed in OpenSSL 1.1.0k (Affected 1.1.0-1.1.0j)."

var nvdValue419 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue418}

var nvdValue420 = schema.CVSSV30{BaseScore: 7.4, VectorString: "CVSS:3.0/AV:N/AC:H/PR:N/UI:N/S:U/C:H/I:H/A:N", Version: "3.0"}

var nvdValue421 = schema.CVEAPIJSON20CVSSV30{CvssData: &nvdValue420}

var nvdValue422 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue95}, CvssMetricV30: []*schema.CVEAPIJSON20CVSSV30{&nvdValue421}}

var nvdValue423 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue419}, ID: "CVE-2019-1543", LastModified: "2026-06-17T02:28:43.290", Metrics: &nvdValue422, Published: "2019-03-06T21:29:00.247"}

var nvdValue424 = "Vixie Cron before the 3.0pl1-133 Debian package allows local users to cause a denial of service (daemon crash) via a large crontab file because the calloc return value is not checked."

var nvdValue425 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue424}

var nvdValue426 = schema.CVSSV31{BaseScore: 5.5, VectorString: "CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:N/I:N/A:H", Version: "3.1"}

var nvdValue427 = schema.CVEAPIJSON20CVSSV31{CvssData: &nvdValue426}

var nvdValue428 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue313}, CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue427}}

var nvdValue429 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue425}, ID: "CVE-2019-9704", LastModified: "2026-06-17T02:44:12.370", Metrics: &nvdValue428, Published: "2019-03-12T01:29:00.240"}

var nvdValue430 = "Vixie Cron before the 3.0pl1-133 Debian package allows local users to cause a denial of service (memory consumption) via a large crontab file because an unlimited number of lines is accepted."

var nvdValue431 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue430}

var nvdValue432 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue431}, ID: "CVE-2019-9705", LastModified: "2026-06-17T02:44:12.490", Metrics: &nvdValue428, Published: "2019-03-12T01:29:00.287"}

var nvdValue433 = "Vixie Cron before the 3.0pl1-133 Debian package allows local users to cause a denial of service (use-after-free and daemon crash) because of a force_rescan_user error."

var nvdValue434 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue433}

var nvdValue435 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue434}, ID: "CVE-2019-9706", LastModified: "2026-06-17T02:44:12.613", Metrics: &nvdValue428, Published: "2019-03-12T01:29:00.317"}

var nvdValue436 = "libxslt through 1.1.33 allows bypass of a protection mechanism because callers of xsltCheckRead and xsltCheckWrite permit access even upon receiving a -1 error code. xsltCheckRead can return -1 for a crafted URL that is not actually invalid and is subsequently loaded."

var nvdValue437 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue436}

var nvdValue438 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue437}, ID: "CVE-2019-11068", LastModified: "2026-06-17T02:12:14.650", Metrics: &nvdValue30, Published: "2019-04-10T20:29:01.147"}

var nvdValue439 = schema.CVEAPIJSON20LangString{Lang: "en", Value: "An integer overflow in curl's URL API results in a buffer overflow in libcurl 7.62.0 to and including 7.64.1."}

var nvdValue440 = schema.CVSSV30{BaseScore: 3.7, VectorString: "CVSS:3.0/AV:N/AC:H/PR:N/UI:N/S:U/C:N/I:N/A:L", Version: "3.0"}

var nvdValue441 = schema.CVEAPIJSON20CVSSV30{CvssData: &nvdValue440}

var nvdValue442 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue214}, CvssMetricV30: []*schema.CVEAPIJSON20CVSSV30{&nvdValue441}}

var nvdValue443 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue439}, ID: "CVE-2019-5435", LastModified: "2026-06-17T02:37:41.007", Metrics: &nvdValue442, Published: "2019-05-28T19:29:06.080"}

var nvdValue444 = "A heap buffer overflow in the TFTP receiving code allows for DoS or arbitrary code execution in libcurl versions 7.19.4 through 7.64.1."

var nvdValue445 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue444}

var nvdValue446 = schema.CVSSV20{BaseScore: 4.6, VectorString: "AV:L/AC:L/Au:N/C:P/I:P/A:P", Version: "2.0"}

var nvdValue447 = schema.CVEAPIJSON20CVSSV2{CvssData: &nvdValue446}

var nvdValue448 = schema.CVSSV31{BaseScore: 7.8, VectorString: "CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:H/I:H/A:H", Version: "3.1"}

var nvdValue449 = schema.CVEAPIJSON20CVSSV31{CvssData: &nvdValue448}

var nvdValue450 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue447}, CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue449}}

var nvdValue451 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue445}, ID: "CVE-2019-5436", LastModified: "2026-06-17T02:37:41.127", Metrics: &nvdValue450, Published: "2019-05-28T19:29:06.127"}

var nvdValue452 = "In Libgcrypt 1.8.4, the C implementation of AES is vulnerable to a flush-and-reload side-channel attack because physical addresses are available to other processes. (The C implementation is used on platforms where an assembly-language implementation is unavailable.) NOTE: the vendor's position is that the issue report cannot be validated because there is no description of an attack"

var nvdValue453 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue452}

var nvdValue454 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue453}, ID: "CVE-2019-12904", LastModified: "2026-06-17T02:15:42.000", Metrics: &nvdValue369, Published: "2019-06-20T00:15:10.667"}

var nvdValue455 = "In numbers.c in libxslt 1.1.33, an xsl:number with certain format strings could lead to a uninitialized read in xsltNumberFormatInsertNumbers. This could allow an attacker to discern whether a byte on the stack contains the characters A, a, I, i, or 0, or any other character."

var nvdValue456 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue455}

var nvdValue457 = schema.CVSSV31{BaseScore: 5.3, VectorString: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:N/A:N", Version: "3.1"}

var nvdValue458 = schema.CVEAPIJSON20CVSSV31{CvssData: &nvdValue457}

var nvdValue459 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue3}, CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue458}}

var nvdValue460 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue456}, ID: "CVE-2019-13117", LastModified: "2026-06-17T02:16:05.863", Metrics: &nvdValue459, Published: "2019-07-01T02:15:09.737"}

var nvdValue461 = "In numbers.c in libxslt 1.1.33, a type holding grouping characters of an xsl:number instruction was too narrow and an invalid character/length combination could be passed to xsltNumberFormatDecimal, leading to a read of uninitialized stack data."

var nvdValue462 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue461}

var nvdValue463 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue462}, ID: "CVE-2019-13118", LastModified: "2026-06-17T02:16:06.150", Metrics: &nvdValue459, Published: "2019-07-01T02:15:09.800"}

var nvdValue464 = "musl libc through 1.1.23 has an x87 floating-point stack adjustment imbalance, related to the math/i386/ directory. In some cases, use of this library could introduce out-of-bounds writes that are not present in an application's source code."

var nvdValue465 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue464}

var nvdValue466 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue465}, ID: "CVE-2019-14697", LastModified: "2026-06-17T02:18:50.250", Metrics: &nvdValue30, Published: "2019-08-06T16:15:11.720"}

var nvdValue467 = "Some HTTP/2 implementations are vulnerable to window size manipulation and stream prioritization manipulation, potentially leading to a denial of service. The attacker requests a large amount of data from a specified resource over multiple streams. They manipulate window size and stream priority to force the server to queue the data in 1-byte chunks. Depending on how efficiently this data is queued, this can consume excess CPU, memory, or both."

var nvdValue468 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue467}

var nvdValue469 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue468}, ID: "CVE-2019-9511", LastModified: "2026-06-17T02:43:51.330", Metrics: &nvdValue378, Published: "2019-08-13T21:15:12.223"}

var nvdValue470 = "Some HTTP/2 implementations are vulnerable to resource loops, potentially leading to a denial of service. The attacker creates multiple request streams and continually shuffles the priority of the streams in a way that causes substantial churn to the priority tree. This can consume excess CPU."

var nvdValue471 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue470}

var nvdValue472 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue471}, ID: "CVE-2019-9513", LastModified: "2026-06-17T02:43:52.087", Metrics: &nvdValue378, Published: "2019-08-13T21:15:12.380"}

var nvdValue473 = "Some HTTP/2 implementations are vulnerable to a header leak, potentially leading to a denial of service. The attacker sends a stream of headers with a 0-length header name and 0-length header value, optionally Huffman encoded into 1-byte or greater headers. Some implementations allocate memory for these headers and keep the allocation alive until the session dies. This can consume excess memory."

var nvdValue474 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue473}

var nvdValue475 = schema.CVSSV20{BaseScore: 6.8, VectorString: "AV:N/AC:L/Au:S/C:N/I:N/A:C", Version: "2.0"}

var nvdValue476 = schema.CVEAPIJSON20CVSSV2{CvssData: &nvdValue475}

var nvdValue477 = schema.CVSSV31{BaseScore: 6.5, VectorString: "CVSS:3.1/AV:N/AC:L/PR:L/UI:N/S:U/C:N/I:N/A:H", Version: "3.1"}

var nvdValue478 = schema.CVEAPIJSON20CVSSV31{CvssData: &nvdValue477}

var nvdValue479 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue476}, CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue478}}

var nvdValue480 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue474}, ID: "CVE-2019-9516", LastModified: "2026-06-17T02:43:53.023", Metrics: &nvdValue479, Published: "2019-08-13T21:15:12.583"}

var nvdValue481 = schema.CVEAPIJSON20LangString{Lang: "en", Value: "Double-free vulnerability in the FTP-kerberos code in cURL 7.52.0 to 7.65.3."}

var nvdValue482 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue481}, ID: "CVE-2019-5481", LastModified: "2026-06-17T02:37:45.923", Metrics: &nvdValue30, Published: "2019-09-16T19:15:10.587"}

var nvdValue483 = schema.CVEAPIJSON20LangString{Lang: "en", Value: "Heap buffer overflow in the TFTP protocol handler in cURL 7.19.4 to 7.65.3."}

var nvdValue484 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue483}, ID: "CVE-2019-5482", LastModified: "2026-06-17T02:37:46.130", Metrics: &nvdValue30, Published: "2019-09-16T19:15:10.633"}

var nvdValue485 = "It was discovered that there was a ECDSA timing attack in the libgcrypt20 cryptographic library. Version affected: 1.8.4-5, 1.7.6-2+deb9u3, and 1.6.3-2+deb8u4. Versions fixed: 1.8.5-2 and 1.6.3-2+deb8u7."

var nvdValue486 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue485}

var nvdValue487 = schema.CVSSV20{BaseScore: 2.6, VectorString: "AV:L/AC:H/Au:N/C:P/I:P/A:N", Version: "2.0"}

var nvdValue488 = schema.CVEAPIJSON20CVSSV2{CvssData: &nvdValue487}

var nvdValue489 = schema.CVSSV31{BaseScore: 6.3, VectorString: "CVSS:3.1/AV:L/AC:H/PR:N/UI:R/S:U/C:H/I:H/A:N", Version: "3.1"}

var nvdValue490 = schema.CVEAPIJSON20CVSSV31{CvssData: &nvdValue489}

var nvdValue491 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue488}, CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue490}}

var nvdValue492 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue486}, ID: "CVE-2019-13627", LastModified: "2026-06-17T02:17:06.567", Metrics: &nvdValue491, Published: "2019-09-25T15:15:11.877"}

var nvdValue493 = "In xsltCopyText in transform.c in libxslt 1.1.33, a pointer variable isn't reset under certain circumstances. If the relevant memory area happened to be freed and reused in a certain way, a bounds check could fail and memory outside a buffer could be written to, or uninitialized data could be disclosed."

var nvdValue494 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue493}

var nvdValue495 = schema.CVSSV20{BaseScore: 5.1, VectorString: "AV:N/AC:H/Au:N/C:P/I:P/A:P", Version: "2.0"}

var nvdValue496 = schema.CVEAPIJSON20CVSSV2{CvssData: &nvdValue495}

var nvdValue497 = schema.CVSSV31{BaseScore: 7.5, VectorString: "CVSS:3.1/AV:N/AC:H/PR:N/UI:R/S:U/C:H/I:H/A:H", Version: "3.1"}

var nvdValue498 = schema.CVEAPIJSON20CVSSV31{CvssData: &nvdValue497}

var nvdValue499 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue496}, CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue498}}

var nvdValue500 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue494}, ID: "CVE-2019-18197", LastModified: "2026-06-17T02:24:27.370", Metrics: &nvdValue499, Published: "2019-10-18T21:15:10.793"}

var nvdValue501 = "In generate_jsimd_ycc_rgb_convert_neon of jsimd_arm64_neon.S, there is a possible out of bounds write due to a missing bounds check. This could lead to remote code execution in an unprivileged process with no additional execution privileges needed. User interaction is needed for exploitation.Product: AndroidVersions: Android-8.0 Android-8.1 Android-9 Android-10Android ID: A-120551338"

var nvdValue502 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue501}

var nvdValue503 = schema.CVSSV31{BaseScore: 7.8, VectorString: "CVSS:3.1/AV:L/AC:L/PR:N/UI:R/S:U/C:H/I:H/A:H", Version: "3.1"}

var nvdValue504 = schema.CVEAPIJSON20CVSSV31{CvssData: &nvdValue503}

var nvdValue505 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue41}, CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue504}}

var nvdValue506 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue502}, ID: "CVE-2019-2201", LastModified: "2026-06-17T02:33:25.853", Metrics: &nvdValue505, Published: "2019-11-13T18:15:11.530"}

var nvdValue507 = "There is an overflow bug in the x64_64 Montgomery squaring procedure used in exponentiation with 512-bit moduli. No EC algorithms are affected. Analysis suggests that attacks against 2-prime RSA1024, 3-prime RSA1536, and DSA1024 as a result of this defect would be very difficult to perform and are not believed likely. Attacks against DH512 are considered just feasible. However, for an attack the target would have to re-use the DH512 private key, which is not recommended anyway. Also applications directly using the low level API BN_mod_exp may be affected if they use BN_FLG_CONSTTIME. Fixed in OpenSSL 1.1.1e (Affected 1.1.1-1.1.1d). Fixed in OpenSSL 1.0.2u (Affected 1.0.2-1.0.2t)."

var nvdValue508 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue507}

var nvdValue509 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue508}, ID: "CVE-2019-1551", LastModified: "2026-06-17T02:28:43.940", Metrics: &nvdValue459, Published: "2019-12-06T18:15:12.840"}

var nvdValue510 = schema.CVEAPIJSON20LangString{Lang: "en", Value: "xmlParseBalancedChunkMemoryRecover in parser.c in libxml2 before 2.9.10 has a memory leak related to newDoc->oldNs."}

var nvdValue511 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue180}, CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue377}}

var nvdValue512 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue510}, ID: "CVE-2019-19956", LastModified: "2026-06-17T02:27:30.643", Metrics: &nvdValue511, Published: "2019-12-24T16:15:11.450"}

var nvdValue513 = "NGINX before 1.17.7, with certain error_page configurations, allows HTTP request smuggling, as demonstrated by the ability of an attacker to read unauthorized web pages in environments where NGINX is being fronted by a load balancer."

var nvdValue514 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue513}

var nvdValue515 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue203}, CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue458}}

var nvdValue516 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue514}, ID: "CVE-2019-20372", LastModified: "2026-06-17T02:30:18.467", Metrics: &nvdValue515, Published: "2020-01-09T21:15:12.027"}

var nvdValue517 = "repodata_schema2id in repodata.c in libsolv before 0.7.6 has a heap-based buffer over-read via a last schema whose length is less than the length of the input schema."

var nvdValue518 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue517}

var nvdValue519 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue518}, ID: "CVE-2019-20387", LastModified: "2026-06-17T02:30:20.330", Metrics: &nvdValue511, Published: "2020-01-21T23:15:13.443"}

var nvdValue520 = "Apache Struts 2.0.0 to 2.5.20 forced double OGNL evaluation, when evaluated on raw user input in tag attributes, may lead to remote code execution."

var nvdValue521 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue520}

var nvdValue522 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue521}, ID: "CVE-2019-0230", LastModified: "2026-06-17T02:08:02.653", Metrics: &nvdValue30, Published: "2020-09-14T17:15:09.933"}

var nvdValue523 = "An access permission override in Apache Struts 2.0.0 to 2.5.20 may cause a Denial of Service when performing a file upload."

var nvdValue524 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue523}

var nvdValue525 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue524}, ID: "CVE-2019-0233", LastModified: "2026-06-17T02:08:03.257", Metrics: &nvdValue511, Published: "2020-09-14T17:15:09.980"}

var nvdValue526 = "Prototype pollution vulnerability in dot-prop npm package versions before 4.2.1 and versions 5.x before 5.1.1 allows an attacker to add arbitrary properties to JavaScript language constructs such as objects."

var nvdValue527 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue526}

var nvdValue528 = schema.CVSSV31{BaseScore: 7.3, VectorString: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:L/A:L", Version: "3.1"}

var nvdValue529 = schema.CVEAPIJSON20CVSSV31{CvssData: &nvdValue528}

var nvdValue530 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue27}, CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue529}}

var nvdValue531 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue527}, ID: "CVE-2020-8116", LastModified: "2026-06-17T03:25:53.353", Metrics: &nvdValue530, Published: "2020-02-04T20:15:13.353"}

var nvdValue532 = schema.CVEAPIJSON20LangString{Lang: "en", Value: "yargs-parser could be tricked into adding or modifying properties of Object.prototype using a \"__proto__\" payload."}

var nvdValue533 = schema.CVSSV31{BaseScore: 5.3, VectorString: "CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:L/I:L/A:L", Version: "3.1"}

var nvdValue534 = schema.CVEAPIJSON20CVSSV31{CvssData: &nvdValue533}

var nvdValue535 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue447}, CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue534}}

var nvdValue536 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue532}, ID: "CVE-2020-7608", LastModified: "2026-06-17T03:25:06.803", Metrics: &nvdValue535, Published: "2020-03-16T20:15:12.860"}

var nvdValue537 = "Server or client applications that call the SSL_check_chain() function during or after a TLS 1.3 handshake may crash due to a NULL pointer dereference as a result of incorrect handling of the \"signature_algorithms_cert\" TLS extension. The crash occurs if an invalid or unrecognised signature algorithm is received from the peer. This could be exploited by a malicious peer in a Denial of Service attack. OpenSSL version 1.1.1d, 1.1.1e, and 1.1.1f are affected by this issue. This issue did not affect OpenSSL versions prior to 1.1.1d. Fixed in OpenSSL 1.1.1g (Affected 1.1.1d-1.1.1f)."

var nvdValue538 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue537}

var nvdValue539 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue538}, ID: "CVE-2020-1967", LastModified: "2026-06-17T03:02:44.283", Metrics: &nvdValue511, Published: "2020-04-21T14:15:11.287"}

var nvdValue540 = "Improper validation of certificate with host mismatch in Apache Log4j SMTP appender. This could allow an SMTPS connection to be intercepted by a man-in-the-middle attack which could leak any log messages sent through that appender. Fixed in Apache Log4j 2.12.3 and 2.13.1"

var nvdValue541 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue540}

var nvdValue542 = schema.CVSSV31{BaseScore: 3.7, VectorString: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:L/I:N/A:N", Version: "3.1"}

var nvdValue543 = schema.CVEAPIJSON20CVSSV31{CvssData: &nvdValue542}

var nvdValue544 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue203}, CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue543}}

var nvdValue545 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue541}, ID: "CVE-2020-9488", LastModified: "2026-06-17T03:28:02.483", Metrics: &nvdValue544, Published: "2020-04-27T16:15:12.897"}

var nvdValue546 = "FasterXML jackson-databind 2.x before 2.9.10.5 mishandles the interaction between serialization gadgets and typing, related to oracle.jms.AQjmsQueueConnectionFactory, oracle.jms.AQjmsXATopicConnectionFactory, oracle.jms.AQjmsTopicConnectionFactory, oracle.jms.AQjmsXAQueueConnectionFactory, and oracle.jms.AQjmsXAConnectionFactory (aka weblogic/oracle-aqjms)."

var nvdValue547 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue546}

var nvdValue548 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue547}, ID: "CVE-2020-14061", LastModified: "2026-08-25T16:28:27.310", Metrics: &nvdValue348, Published: "2020-06-14T20:15:10.027"}

var nvdValue549 = "FasterXML jackson-databind 2.x before 2.9.10.5 mishandles the interaction between serialization gadgets and typing, related to com.sun.org.apache.xalan.internal.lib.sql.JNDIConnectionPool (aka xalan2)."

var nvdValue550 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue549}

var nvdValue551 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue550}, ID: "CVE-2020-14062", LastModified: "2026-08-25T16:28:27.310", Metrics: &nvdValue348, Published: "2020-06-14T20:15:10.167"}

var nvdValue552 = "FasterXML jackson-databind 2.x before 2.9.10.5 mishandles the interaction between serialization gadgets and typing, related to oadd.org.apache.xalan.lib.sql.JNDIConnectionPool (aka apache/drill)."

var nvdValue553 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue552}

var nvdValue554 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue553}, ID: "CVE-2020-14060", LastModified: "2026-08-25T16:28:27.310", Metrics: &nvdValue348, Published: "2020-06-14T21:15:09.817"}

var nvdValue555 = schema.CVEAPIJSON20LangString{Lang: "en", Value: "libpcre in PCRE before 8.44 allows an integer overflow via a large number after a (?C substring."}

var nvdValue556 = schema.CVSSV31{BaseScore: 5.3, VectorString: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:L", Version: "3.1"}

var nvdValue557 = schema.CVEAPIJSON20CVSSV31{CvssData: &nvdValue556}

var nvdValue558 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue180}, CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue557}}

var nvdValue559 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue555}, ID: "CVE-2020-14155", LastModified: "2026-06-17T02:54:18.380", Metrics: &nvdValue558, Published: "2020-06-15T17:15:10.777"}

var nvdValue560 = "FasterXML jackson-databind 2.x before 2.9.10.5 mishandles the interaction between serialization gadgets and typing, related to org.jsecurity.realm.jndi.JndiRealmFactory (aka org.jsecurity)."

var nvdValue561 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue560}

var nvdValue562 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue561}, ID: "CVE-2020-14195", LastModified: "2026-08-25T16:28:27.310", Metrics: &nvdValue348, Published: "2020-06-16T16:15:11.107"}

var nvdValue563 = "Versions of the npm CLI prior to 6.14.6 are vulnerable to an information exposure vulnerability through log files. The CLI supports URLs like \"<protocol>://[<user>[:<password>]@]<hostname>[:<port>][:][/]<path>\". The password value is not redacted and is printed to stdout and also to any generated log files."

var nvdValue564 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue563}

var nvdValue565 = schema.CVSSV31{BaseScore: 4.4, VectorString: "CVSS:3.1/AV:L/AC:H/PR:L/UI:R/S:U/C:H/I:N/A:N", Version: "3.1"}

var nvdValue566 = schema.CVEAPIJSON20CVSSV31{CvssData: &nvdValue565}

var nvdValue567 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue338}, CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue566}}

var nvdValue568 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue564}, ID: "CVE-2020-15095", LastModified: "2026-06-17T02:56:02.600", Metrics: &nvdValue567, Published: "2020-07-07T19:15:10.833"}

var nvdValue569 = "An issue was discovered in ajv.validate() in Ajv (aka Another JSON Schema Validator) 6.12.2. A carefully crafted JSON schema could be provided that allows execution of other code by prototype pollution. (While untrusted schemas are recommended against, the worst case of an untrusted schema should be a denial of service, not execution of code.)"

var nvdValue570 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue569}

var nvdValue571 = schema.CVSSV31{BaseScore: 5.6, VectorString: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:L/I:L/A:L", Version: "3.1"}

var nvdValue572 = schema.CVEAPIJSON20CVSSV31{CvssData: &nvdValue571}

var nvdValue573 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue47}, CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue572}}

var nvdValue574 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue570}, ID: "CVE-2020-15366", LastModified: "2026-06-17T02:56:34.053", Metrics: &nvdValue573, Published: "2020-07-15T20:15:13.380"}

var nvdValue575 = "This affects the package express-fileupload before 1.1.8. If the parseNested option is enabled, sending a corrupt HTTP request can lead to denial of service or arbitrary code execution."

var nvdValue576 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue575}

var nvdValue577 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue576}, ID: "CVE-2020-7699", LastModified: "2026-06-17T03:25:17.297", Metrics: &nvdValue30, Published: "2020-07-30T09:15:11.373"}

var nvdValue578 = "FasterXML jackson-databind 2.x before 2.9.10.6 mishandles the interaction between serialization gadgets and typing, related to br.com.anteros.dbcp.AnterosDBCPDataSource (aka Anteros-DBCP)."

var nvdValue579 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue578}

var nvdValue580 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue579}, ID: "CVE-2020-24616", LastModified: "2026-08-25T16:28:27.310", Metrics: &nvdValue348, Published: "2020-08-25T18:15:11.133"}

var nvdValue581 = "GNOME project libxml2 v2.9.10 has a global buffer over-read vulnerability in xmlEncodeEntitiesInternal at libxml2/entities.c. The issue has been fixed in commit 50f06b3e."

var nvdValue582 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue581}

var nvdValue583 = schema.CVSSV20{BaseScore: 6.4, VectorString: "AV:N/AC:L/Au:N/C:P/I:N/A:P", Version: "2.0"}

var nvdValue584 = schema.CVEAPIJSON20CVSSV2{CvssData: &nvdValue583}

var nvdValue585 = schema.CVSSV31{BaseScore: 6.5, VectorString: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:N/A:L", Version: "3.1"}

var nvdValue586 = schema.CVEAPIJSON20CVSSV31{CvssData: &nvdValue585}

var nvdValue587 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue584}, CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue586}}

var nvdValue588 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue582}, ID: "CVE-2020-24977", LastModified: "2026-06-17T03:06:11.700", Metrics: &nvdValue587, Published: "2020-09-04T00:15:10.693"}

var nvdValue589 = "FasterXML jackson-databind 2.x before 2.9.10.6 mishandles the interaction between serialization gadgets and typing, related to com.pastdev.httpcomponents.configuration.JndiConfiguration."

var nvdValue590 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue589}

var nvdValue591 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue590}, ID: "CVE-2020-24750", LastModified: "2026-08-25T16:28:27.310", Metrics: &nvdValue348, Published: "2020-09-17T19:15:13.580"}

var nvdValue592 = "The implementation of realpath in libuv < 10.22.1, < 12.18.4, and < 14.9.0 used within Node.js incorrectly determined the buffer size which can result in a buffer overflow if the resolved path is longer than 256 bytes."

var nvdValue593 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue592}

var nvdValue594 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue593}, ID: "CVE-2020-8252", LastModified: "2026-06-17T03:26:08.197", Metrics: &nvdValue450, Published: "2020-09-18T21:15:13.497"}

var nvdValue595 = "This affects the package npm-user-validate before 1.0.1. The regex that validates user emails took exponentially longer to process long input strings beginning with @ characters."

var nvdValue596 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue595}

var nvdValue597 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue180}}

var nvdValue598 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue596}, ID: "CVE-2020-7754", LastModified: "2026-06-17T03:25:23.267", Metrics: &nvdValue597, Published: "2020-10-27T15:15:13.123"}

var nvdValue599 = "Heap buffer overflow in Freetype in Google Chrome prior to 86.0.4240.111 allowed a remote attacker to potentially exploit heap corruption via a crafted HTML page."

var nvdValue600 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue599}

var nvdValue601 = schema.CVSSV31{BaseScore: 9.6, VectorString: "CVSS:3.1/AV:N/AC:L/PR:N/UI:R/S:C/C:H/I:H/A:H", Version: "3.1"}

var nvdValue602 = schema.CVEAPIJSON20CVSSV31{CvssData: &nvdValue601}

var nvdValue603 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue214}, CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue602}}

var nvdValue604 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue600}, ID: "CVE-2020-15999", LastModified: "2026-06-17T02:57:36.243", Metrics: &nvdValue603, Published: "2020-11-03T03:15:14.853"}

var nvdValue605 = schema.CVEAPIJSON20LangString{Lang: "en", Value: "The package y18n before 3.2.2, 4.0.1 and 5.0.5, is vulnerable to Prototype Pollution."}

var nvdValue606 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue605}, ID: "CVE-2020-7774", LastModified: "2026-06-17T03:25:25.397", Metrics: &nvdValue30, Published: "2020-11-17T13:15:12.633"}

var nvdValue607 = "In musl libc through 1.2.1, wcsnrtombs mishandles particular combinations of destination buffer size and source character limit, as demonstrated by an invalid write access (buffer overflow)."

var nvdValue608 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue607}

var nvdValue609 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue608}, ID: "CVE-2020-28928", LastModified: "2026-06-17T03:10:50.527", Metrics: &nvdValue428, Published: "2020-11-24T18:15:12.207"}

var nvdValue610 = "A flaw was found in FasterXML Jackson Databind, where it did not have entity expansion secured properly. This flaw allows vulnerability to XML external entity (XXE) attacks. The highest threat from this vulnerability is data integrity."

var nvdValue611 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue610}

var nvdValue612 = schema.CVSSV31{BaseScore: 7.5, VectorString: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:H/A:N", Version: "3.1"}

var nvdValue613 = schema.CVEAPIJSON20CVSSV31{CvssData: &nvdValue612}

var nvdValue614 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue15}, CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue613}}

var nvdValue615 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue611}, ID: "CVE-2020-25649", LastModified: "2026-08-25T16:28:27.310", Metrics: &nvdValue614, Published: "2020-12-03T17:15:12.503"}

var nvdValue616 = "The X.509 GeneralName type is a generic type for representing different types of names. One of those name types is known as EDIPartyName. OpenSSL provides a function GENERAL_NAME_cmp which compares different instances of a GENERAL_NAME to see if they are equal or not. This function behaves incorrectly when both GENERAL_NAMEs contain an EDIPARTYNAME. A NULL pointer dereference and a crash may occur leading to a possible denial of service attack. OpenSSL itself uses the GENERAL_NAME_cmp function for two purposes: 1) Comparing CRL distribution point names between an available CRL and a CRL distribution point embedded in an X509 certificate 2) When verifying that a timestamp response token signer matches the timestamp authority name (exposed via the API functions TS_RESP_verify_response and TS_RESP_verify_token) If an attacker can control both items being compared then that attacker could trigger a crash. For example if the attacker can trick a client or server into checking a malicious certificate against a malicious CRL then this may occur. Note that some applications automatically download CRLs based on a URL embedded in a certificate. This checking happens prior to the signatures on the certificate and CRL being verified. OpenSSL's s_server, s_client and verify tools have support for the \"-crl_download\" option which implements automatic CRL downloading and this attack has been demonstrated to work against those tools. Note that an unrelated bug means that affected versions of OpenSSL cannot parse or construct correct encodings of EDIPARTYNAME. However it is possible to construct a malformed EDIPARTYNAME that OpenSSL's parser will accept and hence trigger this attack. All OpenSSL 1.1.1 and 1.0.2 versions are affected by this issue. Other OpenSSL releases are out of support and have not been checked. Fixed in OpenSSL 1.1.1i (Affected 1.1.1-1.1.1h). Fixed in OpenSSL 1.0.2x (Affected 1.0.2-1.0.2w)."

var nvdValue617 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue616}

var nvdValue618 = schema.CVSSV31{BaseScore: 5.9, VectorString: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:N/I:N/A:H", Version: "3.1"}

var nvdValue619 = schema.CVEAPIJSON20CVSSV31{CvssData: &nvdValue618}

var nvdValue620 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue214}, CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue619}}

var nvdValue621 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue617}, ID: "CVE-2020-1971", LastModified: "2026-06-17T03:02:44.770", Metrics: &nvdValue620, Published: "2020-12-08T16:15:11.730"}

var nvdValue622 = "Forced OGNL evaluation, when evaluated on raw user input in tag attributes, may lead to remote code execution. Affected software : Apache Struts 2.0.0 - Struts 2.5.25."

var nvdValue623 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue622}

var nvdValue624 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue623}, ID: "CVE-2020-17530", LastModified: "2026-06-17T02:59:05.640", Metrics: &nvdValue30, Published: "2020-12-11T02:15:10.883"}

var nvdValue625 = "This affects the package ini before 1.3.6. If an attacker submits a malicious INI file to an application that parses it with ini.parse, they will pollute the prototype on the application. This can be exploited further depending on the context."

var nvdValue626 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue625}

var nvdValue627 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue626}, ID: "CVE-2020-7788", LastModified: "2026-06-17T03:25:26.817", Metrics: &nvdValue30, Published: "2020-12-11T11:15:11.447"}

var nvdValue628 = "curl 7.62.0 through 7.70.0 is vulnerable to an information disclosure vulnerability that can lead to a partial password being leaked over the network and to the DNS server(s)."

var nvdValue629 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue628}

var nvdValue630 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue629}, ID: "CVE-2020-8169", LastModified: "2026-06-17T03:25:58.947", Metrics: &nvdValue270, Published: "2020-12-14T20:15:13.357"}

var nvdValue631 = "curl 7.20.0 through 7.70.0 is vulnerable to improper restriction of names for files and other resources that can lead too overwriting a local file when the -J flag is used."

var nvdValue632 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue631}

var nvdValue633 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue632}, ID: "CVE-2020-8177", LastModified: "2026-06-17T03:25:59.823", Metrics: &nvdValue450, Published: "2020-12-14T20:15:13.497"}

var nvdValue634 = schema.CVEAPIJSON20LangString{Lang: "en", Value: "Due to use of a dangling pointer, libcurl 7.29.0 through 7.71.1 can use the wrong connection when sending data."}

var nvdValue635 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue634}, ID: "CVE-2020-8231", LastModified: "2026-06-17T03:26:05.900", Metrics: &nvdValue270, Published: "2020-12-14T20:15:13.590"}

var nvdValue636 = "A malicious server can use the FTP PASV response to trick curl 7.73.0 and earlier into connecting back to a given IP address and port, and this way potentially make curl extract information about services that are otherwise private and not disclosed, for example doing port scanning and service banner extractions."

var nvdValue637 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue636}

var nvdValue638 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue637}, ID: "CVE-2020-8284", LastModified: "2026-06-17T03:26:11.557", Metrics: &nvdValue544, Published: "2020-12-14T20:15:13.903"}

var nvdValue639 = "curl 7.21.0 to and including 7.73.0 is vulnerable to uncontrolled recursion due to a stack overflow issue in FTP wildcard match parsing."

var nvdValue640 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue639}

var nvdValue641 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue640}, ID: "CVE-2020-8285", LastModified: "2026-06-17T03:26:11.810", Metrics: &nvdValue511, Published: "2020-12-14T20:15:13.983"}

var nvdValue642 = "curl 7.41.0 through 7.73.0 is vulnerable to an improper check for certificate revocation due to insufficient verification of the OCSP response."

var nvdValue643 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue642}

var nvdValue644 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue643}, ID: "CVE-2020-8286", LastModified: "2026-06-17T03:26:13.280", Metrics: &nvdValue614, Published: "2020-12-14T20:15:14.043"}

var nvdValue645 = "FasterXML jackson-databind 2.x before 2.9.10.8 mishandles the interaction between serialization gadgets and typing, related to org.apache.commons.dbcp2.datasources.PerUserPoolDataSource."

var nvdValue646 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue645}

var nvdValue647 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue646}, ID: "CVE-2020-35490", LastModified: "2026-08-25T16:28:27.310", Metrics: &nvdValue348, Published: "2020-12-17T19:15:14.417"}

var nvdValue648 = "FasterXML jackson-databind 2.x before 2.9.10.8 mishandles the interaction between serialization gadgets and typing, related to org.apache.commons.dbcp2.datasources.SharedPoolDataSource."

var nvdValue649 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue648}

var nvdValue650 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue649}, ID: "CVE-2020-35491", LastModified: "2026-08-25T16:28:27.310", Metrics: &nvdValue348, Published: "2020-12-17T19:15:14.480"}

var nvdValue651 = "Node.js versions before 10.23.1, 12.20.1, 14.15.4, 15.5.1 are vulnerable to a use-after-free bug in its TLS implementation. When writing to a TLS enabled socket, node::StreamBase::Write calls node::TLSWrap::DoWrite with a freshly allocated WriteWrap object as first argument. If the DoWrite method does not return an error, this object is passed back to the caller as part of a StreamWriteResult structure. This may be exploited to corrupt memory leading to a Denial of Service or potentially other exploits."

var nvdValue652 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue651}

var nvdValue653 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue652}, ID: "CVE-2020-8265", LastModified: "2026-06-17T03:26:09.627", Metrics: &nvdValue348, Published: "2021-01-06T21:15:14.410"}

var nvdValue654 = "Node.js versions before 10.23.1, 12.20.1, 14.15.4, 15.5.1 allow two copies of a header field in an HTTP request (for example, two Transfer-Encoding header fields). In this case, Node.js identifies the first header field and ignores the second. This can lead to HTTP Request Smuggling."

var nvdValue655 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue654}

var nvdValue656 = schema.CVSSV20{BaseScore: 6.4, VectorString: "AV:N/AC:L/Au:N/C:P/I:P/A:N", Version: "2.0"}

var nvdValue657 = schema.CVEAPIJSON20CVSSV2{CvssData: &nvdValue656}

var nvdValue658 = schema.CVSSV31{BaseScore: 6.5, VectorString: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:L/I:L/A:N", Version: "3.1"}

var nvdValue659 = schema.CVEAPIJSON20CVSSV31{CvssData: &nvdValue658}

var nvdValue660 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue657}, CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue659}}

var nvdValue661 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue655}, ID: "CVE-2020-8287", LastModified: "2026-06-17T03:26:13.610", Metrics: &nvdValue660, Published: "2021-01-06T21:15:14.707"}

var nvdValue662 = "jackson-databind before 2.13.0 allows a Java StackOverflow exception and denial of service via a large depth of nested objects."

var nvdValue663 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue662}

var nvdValue664 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue663}, ID: "CVE-2020-36518", LastModified: "2026-06-17T03:15:38.263", Metrics: &nvdValue511, Published: "2022-03-11T07:15:07.800"}

var nvdValue665 = "Calls to EVP_CipherUpdate, EVP_EncryptUpdate and EVP_DecryptUpdate may overflow the output length argument in some cases where the input length is close to the maximum permissable length for an integer on the platform. In such cases the return value from the function call will be 1 (indicating success), but the output length value will be negative. This could cause applications to behave incorrectly or crash. OpenSSL versions 1.1.1i and below are affected by this issue. Users of these versions should upgrade to OpenSSL 1.1.1j. OpenSSL versions 1.0.2x and below are affected by this issue. However OpenSSL 1.0.2 is out of support and no longer receiving public updates. Premium support customers of OpenSSL 1.0.2 should upgrade to 1.0.2y. Other users should upgrade to 1.1.1j. Fixed in OpenSSL 1.1.1j (Affected 1.1.1-1.1.1i). Fixed in OpenSSL 1.0.2y (Affected 1.0.2-1.0.2x)."

var nvdValue666 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue665}

var nvdValue667 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue666}, ID: "CVE-2021-23840", LastModified: "2026-06-17T03:38:54.593", Metrics: &nvdValue511, Published: "2021-02-16T17:15:13.300"}

var nvdValue668 = "The OpenSSL public API function X509_issuer_and_serial_hash() attempts to create a unique hash value based on the issuer and serial number data contained within an X509 certificate. However it fails to correctly handle any errors that may occur while parsing the issuer field (which might occur if the issuer field is maliciously constructed). This may subsequently result in a NULL pointer deref and a crash leading to a potential denial of service attack. The function X509_issuer_and_serial_hash() is never directly called by OpenSSL itself so applications are only vulnerable if they use this function directly and they use it on certificates that may have been obtained from untrusted sources. OpenSSL versions 1.1.1i and below are affected by this issue. Users of these versions should upgrade to OpenSSL 1.1.1j. OpenSSL versions 1.0.2x and below are affected by this issue. However OpenSSL 1.0.2 is out of support and no longer receiving public updates. Premium support customers of OpenSSL 1.0.2 should upgrade to 1.0.2y. Other users should upgrade to 1.1.1j. Fixed in OpenSSL 1.1.1j (Affected 1.1.1-1.1.1i). Fixed in OpenSSL 1.0.2y (Affected 1.0.2-1.0.2x)."

var nvdValue669 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue668}

var nvdValue670 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue669}, ID: "CVE-2021-23841", LastModified: "2026-06-17T03:38:54.823", Metrics: &nvdValue620, Published: "2021-02-16T17:15:13.377"}

var nvdValue671 = "Node.js before 10.24.0, 12.21.0, 14.16.0, and 15.10.0 is vulnerable to a denial of service attack when too many connection attempts with an 'unknownProtocol' are established. This leads to a leak of file descriptors. If a file descriptor limit is configured on the system, then the server is unable to accept new connections and prevent the process also from opening, e.g. a file. If no file descriptor limit is configured, then this lead to an excessive memory usage and cause the system to run out of memory."

var nvdValue672 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue671}

var nvdValue673 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue672}, ID: "CVE-2021-22883", LastModified: "2026-06-17T03:37:56.640", Metrics: &nvdValue378, Published: "2021-03-03T18:15:14.893"}

var nvdValue674 = "Node.js before 10.24.0, 12.21.0, 14.16.0, and 15.10.0 is vulnerable to DNS rebinding attacks as the whitelist includes “localhost6”. When “localhost6” is not present in /etc/hosts, it is just an ordinary domain that is resolved via DNS, i.e., over network. If the attacker controls the victim's DNS server or can spoof its responses, the DNS rebinding protection can be bypassed by using the “localhost6” domain. As long as the attacker uses the “localhost6” domain, they can still apply the attack described in CVE-2018-7160."

var nvdValue675 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue674}

var nvdValue676 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue675}, ID: "CVE-2021-22884", LastModified: "2026-06-17T03:37:56.800", Metrics: &nvdValue499, Published: "2021-03-03T18:15:14.957"}

var nvdValue677 = "decompress_gunzip.c in BusyBox through 1.32.1 mishandles the error bit on the huft_build result pointer, with a resultant invalid free or segmentation fault, via malformed gzip data."

var nvdValue678 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue677}

var nvdValue679 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue678}, ID: "CVE-2021-28831", LastModified: "2026-06-17T03:46:57.420", Metrics: &nvdValue511, Published: "2021-03-19T05:15:13.150"}

var nvdValue680 = "An OpenSSL TLS server may crash if sent a maliciously crafted renegotiation ClientHello message from a client. If a TLSv1.2 renegotiation ClientHello omits the signature_algorithms extension (where it was present in the initial ClientHello), but includes a signature_algorithms_cert extension then a NULL pointer dereference will result, leading to a crash and a denial of service attack. A server is only vulnerable if it has TLSv1.2 and renegotiation enabled (which is the default configuration). OpenSSL TLS clients are not impacted by this issue. All OpenSSL 1.1.1 versions are affected by this issue. Users of these versions should upgrade to OpenSSL 1.1.1k. OpenSSL 1.0.2 is not impacted by this issue. Fixed in OpenSSL 1.1.1k (Affected 1.1.1-1.1.1j)."

var nvdValue681 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue680}

var nvdValue682 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue681}, ID: "CVE-2021-3449", LastModified: "2026-06-17T04:05:07.327", Metrics: &nvdValue620, Published: "2021-03-25T15:15:13.450"}

var nvdValue683 = "The X509_V_FLAG_X509_STRICT flag enables additional security checks of the certificates present in a certificate chain. It is not set by default. Starting from OpenSSL version 1.1.1h a check to disallow certificates in the chain that have explicitly encoded elliptic curve parameters was added as an additional strict check. An error in the implementation of this check meant that the result of a previous check to confirm that certificates in the chain are valid CA certificates was overwritten. This effectively bypasses the check that non-CA certificates must not be able to issue other certificates. If a \"purpose\" has been configured then there is a subsequent opportunity for checks that the certificate is a valid CA. All of the named \"purpose\" values implemented in libcrypto perform this check. Therefore, where a purpose is set the certificate chain will still be rejected even when the strict flag has been used. A purpose is set by default in libssl client and server certificate verification routines, but it can be overridden or removed by an application. In order to be affected, an application must explicitly set the X509_V_FLAG_X509_STRICT verification flag and either not set a purpose for the certificate verification or, in the case of TLS client or server applications, override the default purpose. OpenSSL versions 1.1.1h and newer are affected by this issue. Users of these versions should upgrade to OpenSSL 1.1.1k. OpenSSL 1.0.2 is not impacted by this issue. Fixed in OpenSSL 1.1.1k (Affected 1.1.1h-1.1.1j)."

var nvdValue684 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue683}

var nvdValue685 = schema.CVSSV31{BaseScore: 7.4, VectorString: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:H/I:H/A:N", Version: "3.1"}

var nvdValue686 = schema.CVEAPIJSON20CVSSV31{CvssData: &nvdValue685}

var nvdValue687 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue95}, CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue686}}

var nvdValue688 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue684}, ID: "CVE-2021-3450", LastModified: "2026-06-17T04:05:07.557", Metrics: &nvdValue687, Published: "2021-03-25T15:15:13.560"}

var nvdValue689 = "curl 7.1.1 to and including 7.75.0 is vulnerable to an \"Exposure of Private Personal Information to an Unauthorized Actor\" by leaking credentials in the HTTP Referer: header. libcurl does not strip off user credentials from the URL when automatically populating the Referer: HTTP request header field in outgoing HTTP requests, and therefore risks leaking sensitive data to the server that is the target of the second HTTP request."

var nvdValue690 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue689}

var nvdValue691 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue690}, ID: "CVE-2021-22876", LastModified: "2026-06-17T03:37:55.703", Metrics: &nvdValue459, Published: "2021-04-01T18:15:12.823"}

var nvdValue692 = "curl 7.63.0 to and including 7.75.0 includes vulnerability that allows a malicious HTTPS proxy to MITM a connection due to bad handling of TLS 1.3 session tickets. When using a HTTPS proxy and TLS 1.3, libcurl can confuse session tickets arriving from the HTTPS proxy but work as if they arrived from the remote server and then wrongly \"short-cut\" the host handshake. When confusing the tickets, a HTTPS proxy can trick libcurl to use the wrong session ticket resume for the host and thereby circumvent the server TLS certificate check and make a MITM attack to be possible to perform unnoticed. Note that such a malicious HTTPS proxy needs to provide a certificate that curl will accept for the MITMed server for an attack to work - unless curl has been told to ignore the server certificate check."

var nvdValue693 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue692}

var nvdValue694 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue693}, ID: "CVE-2021-22890", LastModified: "2026-06-17T03:37:57.537", Metrics: &nvdValue36, Published: "2021-04-01T18:15:12.917"}

var nvdValue695 = schema.CVEAPIJSON20LangString{Lang: "en", Value: "In Alpine Linux apk-tools before 2.12.5, the tarball parser allows a buffer overflow and crash."}

var nvdValue696 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue695}, ID: "CVE-2021-30139", LastModified: "2026-06-17T03:49:43.980", Metrics: &nvdValue511, Published: "2021-04-21T16:15:08.830"}

var nvdValue697 = "Apache Maven will follow repositories that are defined in a dependency’s Project Object Model (pom) which may be surprising to some users, resulting in potential risk if a malicious actor takes over that repository or is able to insert themselves into a position to pretend to be that repository. Maven is changing the default behavior in 3.8.1+ to no longer follow http (non-SSL) repository references by default. More details available in the referenced urls. If you are currently using a repository manager to govern the repositories used by your builds, you are unaffected by the risks present in the legacy behavior, and are unaffected by this vulnerability and change to default behavior. See this link for more information about repository management: https://maven.apache.org/repository-management.html"

var nvdValue698 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue697}

var nvdValue699 = schema.CVSSV31{BaseScore: 9.1, VectorString: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:N", Version: "3.1"}

var nvdValue700 = schema.CVEAPIJSON20CVSSV31{CvssData: &nvdValue699}

var nvdValue701 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue657}, CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue700}}

var nvdValue702 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue698}, ID: "CVE-2021-26291", LastModified: "2026-06-17T03:43:03.490", Metrics: &nvdValue701, Published: "2021-04-23T15:15:09.387"}

var nvdValue703 = "Buffer overflow vulnerability in libsolv 2020-12-13 via the Solver * testcase_read(Pool *pool, FILE *fp, const char *testcase, Queue *job, char **resultp, int *resultflagsp function at src/testcase.c: line 2334, which could cause a denial of service"

var nvdValue704 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue703}

var nvdValue705 = schema.CVSSV31{BaseScore: 3.3, VectorString: "CVSS:3.1/AV:L/AC:L/PR:N/UI:R/S:U/C:N/I:N/A:L", Version: "3.1"}

var nvdValue706 = schema.CVEAPIJSON20CVSSV31{CvssData: &nvdValue705}

var nvdValue707 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue214}, CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue706}}

var nvdValue708 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue704}, ID: "CVE-2021-3200", LastModified: "2026-06-17T04:04:50.130", Metrics: &nvdValue707, Published: "2021-05-18T17:15:07.297"}

var nvdValue709 = "There's a flaw in lz4. An attacker who submits a crafted file to an application linked with lz4 may be able to trigger an integer overflow, leading to calling of memmove() on a negative size argument, causing an out-of-bounds write and/or a crash. The greatest impact of this flaw is to availability, with some potential impact to confidentiality and integrity as well."

var nvdValue710 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue709}

var nvdValue711 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue710}, ID: "CVE-2021-3520", LastModified: "2026-06-17T04:05:17.183", Metrics: &nvdValue30, Published: "2021-06-02T13:15:13.170"}

var nvdValue712 = "Libgcrypt before 1.8.8 and 1.9.x before 1.9.3 mishandles ElGamal encryption because it lacks exponent blinding to address a side-channel attack against mpi_powm, and the window size is not chosen appropriately. This, for example, affects use of ElGamal in OpenPGP."

var nvdValue713 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue712}

var nvdValue714 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue713}, ID: "CVE-2021-33560", LastModified: "2026-06-17T03:54:47.733", Metrics: &nvdValue270, Published: "2021-06-08T11:15:07.767"}

var nvdValue715 = "curl 7.7 through 7.76.1 suffers from an information disclosure when the `-t` command line option, known as `CURLOPT_TELNETOPTIONS` in libcurl, is used to send variable=content pairs to TELNET servers. Due to a flaw in the option parser for sending NEW_ENV variables, libcurl could be made to pass on uninitialized data from a stack based buffer to the server, resulting in potentially revealing sensitive internal information to the server using a clear-text network protocol."

var nvdValue716 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue715}

var nvdValue717 = schema.CVSSV20{BaseScore: 2.6, VectorString: "AV:N/AC:H/Au:N/C:P/I:N/A:N", Version: "2.0"}

var nvdValue718 = schema.CVEAPIJSON20CVSSV2{CvssData: &nvdValue717}

var nvdValue719 = schema.CVSSV31{BaseScore: 3.1, VectorString: "CVSS:3.1/AV:N/AC:H/PR:N/UI:R/S:U/C:L/I:N/A:N", Version: "3.1"}

var nvdValue720 = schema.CVEAPIJSON20CVSSV31{CvssData: &nvdValue719}

var nvdValue721 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue718}, CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue720}}

var nvdValue722 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue716}, ID: "CVE-2021-22898", LastModified: "2026-06-17T03:37:58.730", Metrics: &nvdValue721, Published: "2021-06-11T16:15:11.043"}

var nvdValue723 = "basic/unit-name.c in systemd prior to 246.15, 247.8, 248.5, and 249.1 has a Memory Allocation with an Excessive Size Value (involving strdupa and alloca for a pathname controlled by a local attacker) that results in an operating system crash."

var nvdValue724 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue723}

var nvdValue725 = schema.CVSSV20{BaseScore: 4.9, VectorString: "AV:L/AC:L/Au:N/C:N/I:N/A:C", Version: "2.0"}

var nvdValue726 = schema.CVEAPIJSON20CVSSV2{CvssData: &nvdValue725}

var nvdValue727 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue726}, CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue427}}

var nvdValue728 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue724}, ID: "CVE-2021-33910", LastModified: "2026-06-17T03:55:21.827", Metrics: &nvdValue727, Published: "2021-07-20T19:15:09.783"}

var nvdValue729 = "libfetch before 2021-07-26, as used in apk-tools, xbps, and other products, mishandles numeric strings for the FTP and HTTP protocols. The FTP passive mode implementation allows an out-of-bounds read because strtol is used to parse the relevant numbers into address bytes. It does not check if the line ends prematurely. If it does, the for-loop condition checks for the '\\0' terminator one byte too late."

var nvdValue730 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue729}

var nvdValue731 = schema.CVSSV31{BaseScore: 9.1, VectorString: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:N/A:H", Version: "3.1"}

var nvdValue732 = schema.CVEAPIJSON20CVSSV31{CvssData: &nvdValue731}

var nvdValue733 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue584}, CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue732}}

var nvdValue734 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue730}, ID: "CVE-2021-36159", LastModified: "2026-06-17T03:58:26.080", Metrics: &nvdValue733, Published: "2021-08-03T14:15:08.233"}

var nvdValue735 = "libcurl keeps previously used connections in a connection pool for subsequenttransfers to reuse, if one of them matches the setup.Due to errors in the logic, the config matching function did not take 'issuercert' into account and it compared the involved paths *case insensitively*,which could lead to libcurl reusing wrong connections.File paths are, or can be, case sensitive on many systems but not all, and caneven vary depending on used file systems.The comparison also didn't include the 'issuer cert' which a transfer can setto qualify how to verify the server certificate."

var nvdValue736 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue735}

var nvdValue737 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue736}, ID: "CVE-2021-22924", LastModified: "2026-06-17T03:38:02.147", Metrics: &nvdValue544, Published: "2021-08-05T21:15:11.380"}

var nvdValue738 = "curl supports the `-t` command line option, known as `CURLOPT_TELNETOPTIONS`in libcurl. This rarely used option is used to send variable=content pairs toTELNET servers.Due to flaw in the option parser for sending `NEW_ENV` variables, libcurlcould be made to pass on uninitialized data from a stack based buffer to theserver. Therefore potentially revealing sensitive internal information to theserver using a clear-text network protocol.This could happen because curl did not call and use sscanf() correctly whenparsing the string provided by the application."

var nvdValue739 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue738}

var nvdValue740 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue739}, ID: "CVE-2021-22925", LastModified: "2026-06-17T03:38:02.427", Metrics: &nvdValue459, Published: "2021-08-05T21:15:11.467"}

var nvdValue741 = "In order to decrypt SM2 encrypted data an application is expected to call the API function EVP_PKEY_decrypt(). Typically an application will call this function twice. The first time, on entry, the \"out\" parameter can be NULL and, on exit, the \"outlen\" parameter is populated with the buffer size required to hold the decrypted plaintext. The application can then allocate a sufficiently sized buffer and call EVP_PKEY_decrypt() again, but this time passing a non-NULL value for the \"out\" parameter. A bug in the implementation of the SM2 decryption code means that the calculation of the buffer size required to hold the plaintext returned by the first call to EVP_PKEY_decrypt() can be smaller than the actual size required by the second call. This can lead to a buffer overflow when EVP_PKEY_decrypt() is called by the application a second time with a buffer that is too small. A malicious attacker who is able present SM2 content for decryption to an application could cause attacker chosen data to overflow the buffer by up to a maximum of 62 bytes altering the contents of other data held after the buffer, possibly changing application behaviour or causing the application to crash. The location of the buffer is application dependent but is typically heap allocated. Fixed in OpenSSL 1.1.1l (Affected 1.1.1-1.1.1k)."

var nvdValue742 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue741}

var nvdValue743 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue742}, ID: "CVE-2021-3711", LastModified: "2026-06-17T04:05:36.467", Metrics: &nvdValue30, Published: "2021-08-24T15:15:09.133"}

var nvdValue744 = "ASN.1 strings are represented internally within OpenSSL as an ASN1_STRING structure which contains a buffer holding the string data and a field holding the buffer length. This contrasts with normal C strings which are repesented as a buffer for the string data which is terminated with a NUL (0) byte. Although not a strict requirement, ASN.1 strings that are parsed using OpenSSL's own \"d2i\" functions (and other similar parsing functions) as well as any string whose value has been set with the ASN1_STRING_set() function will additionally NUL terminate the byte array in the ASN1_STRING structure. However, it is possible for applications to directly construct valid ASN1_STRING structures which do not NUL terminate the byte array by directly setting the \"data\" and \"length\" fields in the ASN1_STRING array. This can also happen by using the ASN1_STRING_set0() function. Numerous OpenSSL functions that print ASN.1 data have been found to assume that the ASN1_STRING byte array will be NUL terminated, even though this is not guaranteed for strings that have been directly constructed. Where an application requests an ASN.1 structure to be printed, and where that ASN.1 structure contains ASN1_STRINGs that have been directly constructed by the application without NUL terminating the \"data\" field, then a read buffer overrun can occur. The same thing can also occur during name constraints processing of certificates (for example if a certificate has been directly constructed by the application instead of loading it via the OpenSSL parsing functions, and the certificate contains non NUL terminated ASN1_STRING structures). It can also occur in the X509_get1_email(), X509_REQ_get1_email() and X509_get1_ocsp() functions. If a malicious actor can cause an application to directly construct an ASN1_STRING and then process it through one of the affected OpenSSL functions then this issue could be hit. This might result in a crash (causing a Denial of Service attack). It could also result in the disclosure of private memory contents (such as private keys, or sensitive plaintext). Fixed in OpenSSL 1.1.1l (Affected 1.1.1-1.1.1k). Fixed in OpenSSL 1.0.2za (Affected 1.0.2-1.0.2y)."

var nvdValue745 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue744}

var nvdValue746 = schema.CVSSV31{BaseScore: 7.4, VectorString: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:H/I:N/A:H", Version: "3.1"}

var nvdValue747 = schema.CVEAPIJSON20CVSSV31{CvssData: &nvdValue746}

var nvdValue748 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue386}, CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue747}}

var nvdValue749 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue745}, ID: "CVE-2021-3712", LastModified: "2026-06-17T04:05:36.623", Metrics: &nvdValue748, Published: "2021-08-24T15:15:09.533"}

var nvdValue750 = "Buffer overflow vulnerability in function pool_installable in src/repo.h in libsolv before 0.7.17 allows attackers to cause a Denial of Service."

var nvdValue751 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue750}

var nvdValue752 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue751}, ID: "CVE-2021-33928", LastModified: "2026-06-17T03:55:22.963", Metrics: &nvdValue511, Published: "2021-09-02T15:15:07.657"}

var nvdValue753 = "Buffer overflow vulnerability in function pool_disabled_solvable in src/repo.h in libsolv before 0.7.17 allows attackers to cause a Denial of Service."

var nvdValue754 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue753}

var nvdValue755 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue754}, ID: "CVE-2021-33929", LastModified: "2026-06-17T03:55:23.073", Metrics: &nvdValue511, Published: "2021-09-02T15:15:07.707"}

var nvdValue756 = "Buffer overflow vulnerability in function pool_installable_whatprovides in src/repo.h in libsolv before 0.7.17 allows attackers to cause a Denial of Service."

var nvdValue757 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue756}

var nvdValue758 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue757}, ID: "CVE-2021-33930", LastModified: "2026-06-17T03:55:23.183", Metrics: &nvdValue511, Published: "2021-09-02T15:15:07.757"}

var nvdValue759 = "Buffer overflow vulnerability in function prune_to_recommended in src/policy.c in libsolv before 0.7.17 allows attackers to cause a Denial of Service."

var nvdValue760 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue759}

var nvdValue761 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue760}, ID: "CVE-2021-33938", LastModified: "2026-06-17T03:55:23.293", Metrics: &nvdValue511, Published: "2021-09-02T15:15:07.803"}

var nvdValue762 = "The ElGamal implementation in Libgcrypt before 1.9.4 allows plaintext recovery because, during interaction between two cryptographic libraries, a certain dangerous combination of the prime defined by the receiver's public key, the generator defined by the receiver's public key, and the sender's ephemeral exponents can lead to a cross-configuration attack against OpenPGP."

var nvdValue763 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue762}

var nvdValue764 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue718}, CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue368}}

var nvdValue765 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue763}, ID: "CVE-2021-40528", LastModified: "2026-06-17T04:07:05.660", Metrics: &nvdValue764, Published: "2021-09-06T19:15:07.587"}

var nvdValue766 = "A user can tell curl >= 7.20.0 and <= 7.78.0 to require a successful upgrade to TLS when speaking to an IMAP, POP3 or FTP server (`--ssl-reqd` on the command line or`CURLOPT_USE_SSL` set to `CURLUSESSL_CONTROL` or `CURLUSESSL_ALL` withlibcurl). This requirement could be bypassed if the server would return a properly crafted but perfectly legitimate response.This flaw would then make curl silently continue its operations **withoutTLS** contrary to the instructions and expectations, exposing possibly sensitive data in clear text over the network."

var nvdValue767 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue766}

var nvdValue768 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue767}, ID: "CVE-2021-22946", LastModified: "2026-06-17T03:38:05.010", Metrics: &nvdValue270, Published: "2021-09-29T20:15:08.187"}

var nvdValue769 = "When curl >= 7.20.0 and <= 7.78.0 connects to an IMAP or POP3 server to retrieve data using STARTTLS to upgrade to TLS security, the server can respond and send back multiple responses at once that curl caches. curl would then upgrade to TLS but not flush the in-queue of cached responses but instead continue using and trustingthe responses it got *before* the TLS handshake as if they were authenticated.Using this flaw, it allows a Man-In-The-Middle attacker to first inject the fake responses, then pass-through the TLS traffic from the legitimate server and trick curl into sending data back to the user thinking the attacker's injected data comes from the TLS-protected server."

var nvdValue770 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue769}

var nvdValue771 = schema.CVSSV31{BaseScore: 5.9, VectorString: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:N/I:H/A:N", Version: "3.1"}

var nvdValue772 = schema.CVEAPIJSON20CVSSV31{CvssData: &nvdValue771}

var nvdValue773 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue9}, CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue772}}

var nvdValue774 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue770}, ID: "CVE-2021-22947", LastModified: "2026-06-17T03:38:05.227", Metrics: &nvdValue773, Published: "2021-09-29T20:15:08.253"}

var nvdValue775 = "The parse function in llhttp < 2.1.4 and < 6.0.6. ignores chunk extensions when parsing the body of chunked requests. This leads to HTTP Request Smuggling (HRS) under certain conditions."

var nvdValue776 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue775}

var nvdValue777 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue95}, CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue659}}

var nvdValue778 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue776}, ID: "CVE-2021-22960", LastModified: "2026-06-17T03:38:06.730", Metrics: &nvdValue777, Published: "2021-11-03T20:15:08.247"}

var nvdValue779 = "The npm ci command in npm 7.x and 8.x through 8.1.3 proceeds with an installation even if dependency information in package-lock.json differs from package.json. This behavior is inconsistent with the documentation, and makes it easier for attackers to install malware that was supposed to have been blocked by an exact version match requirement in package-lock.json. NOTE: The npm team believes this is not a vulnerability. It would require someone to socially engineer package.json which has different dependencies than package-lock.json. That user would have to have file system or write access to change dependencies. The npm team states preventing malicious actors from socially engineering or gaining file system access is outside the scope of the npm CLI."

var nvdValue780 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue779}

var nvdValue781 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue780}, ID: "CVE-2021-43616", LastModified: "2026-06-17T04:11:12.010", Metrics: &nvdValue30, Published: "2021-11-13T18:15:07.537"}

var nvdValue782 = "The parser in accepts requests with a space (SP) right after the header name before the colon. This can lead to HTTP Request Smuggling (HRS) in llhttp < v2.1.4 and < v6.0.6."

var nvdValue783 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue782}

var nvdValue784 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue783}, ID: "CVE-2021-22959", LastModified: "2026-06-17T03:38:06.607", Metrics: &nvdValue660, Published: "2021-11-15T15:15:06.747"}

var nvdValue785 = "Apache Log4j2 versions 2.0-alpha1 through 2.16.0 (excluding 2.12.3 and 2.3.1) did not protect from uncontrolled recursion from self-referential lookups. This allows an attacker with control over Thread Context Map data to cause a denial of service when a crafted string is interpreted. This issue was fixed in Log4j 2.17.0, 2.12.3, and 2.3.1."

var nvdValue786 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue785}

var nvdValue787 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue786}, ID: "CVE-2021-45105", LastModified: "2026-08-25T16:28:27.310", Metrics: &nvdValue620, Published: "2021-12-18T12:15:07.433"}

var nvdValue788 = "Apache Log4j2 versions 2.0-beta7 through 2.17.0 (excluding security fix releases 2.3.2 and 2.12.4) are vulnerable to a remote code execution (RCE) attack when a configuration uses a JDBC Appender with a JNDI LDAP data source URI when an attacker has control of the target LDAP server. This issue is fixed by limiting JNDI data source names to the java protocol in Log4j2 versions 2.17.1, 2.12.4, and 2.3.2."

var nvdValue789 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue788}

var nvdValue790 = schema.CVSSV20{BaseScore: 8.5, VectorString: "AV:N/AC:M/Au:S/C:C/I:C/A:C", Version: "2.0"}

var nvdValue791 = schema.CVEAPIJSON20CVSSV2{CvssData: &nvdValue790}

var nvdValue792 = schema.CVSSV31{BaseScore: 6.6, VectorString: "CVSS:3.1/AV:N/AC:H/PR:H/UI:N/S:U/C:H/I:H/A:H", Version: "3.1"}

var nvdValue793 = schema.CVEAPIJSON20CVSSV31{CvssData: &nvdValue792}

var nvdValue794 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue791}, CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue793}}

var nvdValue795 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue789}, ID: "CVE-2021-44832", LastModified: "2026-06-17T04:12:52.013", Metrics: &nvdValue794, Published: "2021-12-28T20:15:08.400"}

var nvdValue796 = "In Expat (aka libexpat) before 2.4.3, a left shift by 29 (or more) places in the storeAtts function in xmlparse.c can lead to realloc misbehavior (e.g., allocating too few bytes, or only freeing memory)."

var nvdValue797 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue796}

var nvdValue798 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue150}, CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue70}}

var nvdValue799 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue797}, ID: "CVE-2021-45960", LastModified: "2026-06-17T04:14:19.917", Metrics: &nvdValue798, Published: "2022-01-01T19:15:08.030"}

var nvdValue800 = schema.CVEAPIJSON20LangString{Lang: "en", Value: "In doProlog in xmlparse.c in Expat (aka libexpat) before 2.4.3, an integer overflow exists for m_groupSize."}

var nvdValue801 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue47}, CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue504}}

var nvdValue802 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue800}, ID: "CVE-2021-46143", LastModified: "2026-06-17T04:14:34.540", Metrics: &nvdValue801, Published: "2022-01-06T04:15:07.017"}

var nvdValue803 = "Accepting arbitrary Subject Alternative Name (SAN) types, unless a PKI is specifically defined to use a particular SAN type, can result in bypassing name-constrained intermediates. Node.js < 12.22.9, < 14.18.3, < 16.13.2, and < 17.3.1 was accepting URI SAN types, which PKIs are often not defined to use. Additionally, when a protocol allows URI SANs, Node.js did not match the URI correctly.Versions of Node.js with the fix for this disable the URI SAN type when checking a certificate against a hostname. This behavior can be reverted through the --security-revert command-line option."

var nvdValue804 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue803}

var nvdValue805 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue804}, ID: "CVE-2021-44531", LastModified: "2026-06-17T04:12:31.650", Metrics: &nvdValue687, Published: "2022-02-24T19:15:09.313"}

var nvdValue806 = "Node.js < 12.22.9, < 14.18.3, < 16.13.2, and < 17.3.1 converts SANs (Subject Alternative Names) to a string format. It uses this string to check peer certificates against hostnames when validating connections. The string format was subject to an injection vulnerability when name constraints were used within a certificate chain, allowing the bypass of these name constraints.Versions of Node.js with the fix for this escape SANs containing the problematic characters in order to prevent the injection. This behavior can be reverted through the --security-revert command-line option."

var nvdValue807 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue806}

var nvdValue808 = schema.CVSSV31{BaseScore: 5.3, VectorString: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:L/A:N", Version: "3.1"}

var nvdValue809 = schema.CVEAPIJSON20CVSSV31{CvssData: &nvdValue808}

var nvdValue810 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue15}, CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue809}}

var nvdValue811 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue807}, ID: "CVE-2021-44532", LastModified: "2026-06-17T04:12:31.783", Metrics: &nvdValue810, Published: "2022-02-24T19:15:09.360"}

var nvdValue812 = "Node.js < 12.22.9, < 14.18.3, < 16.13.2, and < 17.3.1 did not handle multi-value Relative Distinguished Names correctly. Attackers could craft certificate subjects containing a single-value Relative Distinguished Name that would be interpreted as a multi-value Relative Distinguished Name, for example, in order to inject a Common Name that would allow bypassing the certificate subject verification.Affected versions of Node.js that do not accept multi-value Relative Distinguished Names and are thus not vulnerable to such attacks themselves. However, third-party code that uses node's ambiguous presentation of certificate subjects may be vulnerable."

var nvdValue813 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue812}

var nvdValue814 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue813}, ID: "CVE-2021-44533", LastModified: "2026-06-17T04:12:31.910", Metrics: &nvdValue810, Published: "2022-02-24T19:15:09.407"}

var nvdValue815 = "The fix issued for CVE-2020-17530 was incomplete. So from Apache Struts 2.0.0 to 2.5.29, still some of the tag’s attributes could perform a double evaluation if a developer applied forced OGNL evaluation by using the %{...} syntax. Using forced OGNL evaluation on untrusted user input can lead to a Remote Code Execution and security degradation."

var nvdValue816 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue815}

var nvdValue817 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue816}, ID: "CVE-2021-31805", LastModified: "2026-06-17T03:52:15.950", Metrics: &nvdValue30, Published: "2022-04-12T16:15:08.133"}

var nvdValue818 = "drools <=7.59.x is affected by an XML External Entity (XXE) vulnerability in KieModuleMarshaller.java. The Validator class is not used correctly, resulting in the XXE injection vulnerability."

var nvdValue819 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue818}

var nvdValue820 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue819}, ID: "CVE-2021-41411", LastModified: "2026-06-17T04:08:28.993", Metrics: &nvdValue30, Published: "2022-06-16T10:15:09.007"}

var nvdValue821 = schema.CVEAPIJSON20LangString{Lang: "en", Value: "addBinding in xmlparse.c in Expat (aka libexpat) before 2.4.3 has an integer overflow."}

var nvdValue822 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue821}, ID: "CVE-2022-22822", LastModified: "2026-06-17T04:29:06.123", Metrics: &nvdValue30, Published: "2022-01-10T14:12:56.047"}

var nvdValue823 = schema.CVEAPIJSON20LangString{Lang: "en", Value: "build_model in xmlparse.c in Expat (aka libexpat) before 2.4.3 has an integer overflow."}

var nvdValue824 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue823}, ID: "CVE-2022-22823", LastModified: "2026-06-17T04:29:06.307", Metrics: &nvdValue30, Published: "2022-01-10T14:12:56.270"}

var nvdValue825 = schema.CVEAPIJSON20LangString{Lang: "en", Value: "defineAttribute in xmlparse.c in Expat (aka libexpat) before 2.4.3 has an integer overflow."}

var nvdValue826 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue825}, ID: "CVE-2022-22824", LastModified: "2026-06-17T04:29:06.497", Metrics: &nvdValue30, Published: "2022-01-10T14:12:56.567"}

var nvdValue827 = schema.CVEAPIJSON20LangString{Lang: "en", Value: "lookup in xmlparse.c in Expat (aka libexpat) before 2.4.3 has an integer overflow."}

var nvdValue828 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue827}, ID: "CVE-2022-22825", LastModified: "2026-06-17T04:29:06.680", Metrics: &nvdValue247, Published: "2022-01-10T14:12:56.847"}

var nvdValue829 = schema.CVEAPIJSON20LangString{Lang: "en", Value: "nextScaffoldPart in xmlparse.c in Expat (aka libexpat) before 2.4.3 has an integer overflow."}

var nvdValue830 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue829}, ID: "CVE-2022-22826", LastModified: "2026-06-17T04:29:06.860", Metrics: &nvdValue247, Published: "2022-01-10T14:12:57.113"}

var nvdValue831 = schema.CVEAPIJSON20LangString{Lang: "en", Value: "storeAtts in xmlparse.c in Expat (aka libexpat) before 2.4.3 has an integer overflow."}

var nvdValue832 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue831}, ID: "CVE-2022-22827", LastModified: "2026-06-17T04:29:07.040", Metrics: &nvdValue247, Published: "2022-01-10T14:12:57.363"}

var nvdValue833 = "Expat (aka libexpat) before 2.4.4 has a signed integer overflow in XML_GetBuffer, for configurations with a nonzero XML_CONTEXT_BYTES."

var nvdValue834 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue833}

var nvdValue835 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue834}, ID: "CVE-2022-23852", LastModified: "2026-06-17T04:30:53.100", Metrics: &nvdValue30, Published: "2022-01-24T02:15:06.733"}

var nvdValue836 = schema.CVEAPIJSON20LangString{Lang: "en", Value: "Expat (aka libexpat) before 2.4.4 has an integer overflow in the doProlog function."}

var nvdValue837 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue836}, ID: "CVE-2022-23990", LastModified: "2026-06-17T04:31:06.400", Metrics: &nvdValue511, Published: "2022-01-26T19:15:08.517"}

var nvdValue838 = "xmltok_impl.c in Expat (aka libexpat) before 2.4.5 lacks certain validation of encoding, such as checks for whether a UTF-8 character is valid in a certain context."

var nvdValue839 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue838}

var nvdValue840 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue839}, ID: "CVE-2022-25235", LastModified: "2026-06-17T04:33:15.017", Metrics: &nvdValue30, Published: "2022-02-16T01:15:07.607"}

var nvdValue841 = "xmlparse.c in Expat (aka libexpat) before 2.4.5 allows attackers to insert namespace-separator characters into namespace URIs."

var nvdValue842 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue841}

var nvdValue843 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue842}, ID: "CVE-2022-25236", LastModified: "2026-06-17T04:33:15.223", Metrics: &nvdValue30, Published: "2022-02-16T01:15:07.650"}

var nvdValue844 = "In Expat (aka libexpat) before 2.4.5, an attacker can trigger stack exhaustion in build_model via a large nesting depth in the DTD element."

var nvdValue845 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue844}

var nvdValue846 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue845}, ID: "CVE-2022-25313", LastModified: "2026-06-17T04:33:22.443", Metrics: &nvdValue354, Published: "2022-02-18T05:15:08.130"}

var nvdValue847 = schema.CVEAPIJSON20LangString{Lang: "en", Value: "In Expat (aka libexpat) before 2.4.5, there is an integer overflow in copyString."}

var nvdValue848 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue847}, ID: "CVE-2022-25314", LastModified: "2026-06-17T04:33:22.650", Metrics: &nvdValue511, Published: "2022-02-18T05:15:08.187"}

var nvdValue849 = schema.CVEAPIJSON20LangString{Lang: "en", Value: "In Expat (aka libexpat) before 2.4.5, there is an integer overflow in storeRawNames."}

var nvdValue850 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue849}, ID: "CVE-2022-25315", LastModified: "2026-06-17T04:33:22.843", Metrics: &nvdValue30, Published: "2022-02-18T05:15:08.237"}

var nvdValue851 = "Due to the formatting logic of the \"console.table()\" function it was not safe to allow user controlled input to be passed to the \"properties\" parameter while simultaneously passing a plain object with at least one property as the first parameter, which could be \"__proto__\". The prototype pollution has very limited control, in that it only allows an empty string to be assigned to numerical keys of the object prototype.Node.js >= 12.22.9, >= 14.18.3, >= 16.13.2, and >= 17.3.1 use a null protoype for the object these properties are being assigned to."

var nvdValue852 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue851}

var nvdValue853 = schema.CVSSV31{BaseScore: 8.2, VectorString: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:L/A:H", Version: "3.1"}

var nvdValue854 = schema.CVEAPIJSON20CVSSV31{CvssData: &nvdValue853}

var nvdValue855 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue53}, CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue854}}

var nvdValue856 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue852}, ID: "CVE-2022-21824", LastModified: "2026-06-17T04:27:03.220", Metrics: &nvdValue855, Published: "2022-02-24T19:15:10.080"}

var nvdValue857 = "The BN_mod_sqrt() function, which computes a modular square root, contains a bug that can cause it to loop forever for non-prime moduli. Internally this function is used when parsing certificates that contain elliptic curve public keys in compressed form or explicit elliptic curve parameters with a base point encoded in compressed form. It is possible to trigger the infinite loop by crafting a certificate that has invalid explicit curve parameters. Since certificate parsing happens prior to verification of the certificate signature, any process that parses an externally supplied certificate may thus be subject to a denial of service attack. The infinite loop can also be reached when parsing crafted private keys as they can contain explicit elliptic curve parameters. Thus vulnerable situations include: - TLS clients consuming server certificates - TLS servers consuming client certificates - Hosting providers taking certificates or private keys from customers - Certificate authorities parsing certification requests from subscribers - Anything else which parses ASN.1 elliptic curve parameters Also any other applications that use the BN_mod_sqrt() where the attacker can control the parameter values are vulnerable to this DoS issue. In the OpenSSL 1.0.2 version the public key is not parsed during initial parsing of the certificate which makes it slightly harder to trigger the infinite loop. However any operation which requires the public key from the certificate will trigger the infinite loop. In particular the attacker can use a self-signed certificate to trigger the loop during verification of the certificate signature. This issue affects OpenSSL versions 1.0.2, 1.1.1 and 3.0. It was addressed in the releases of 1.1.1n and 3.0.2 on the 15th March 2022. Fixed in OpenSSL 3.0.2 (Affected 3.0.0,3.0.1). Fixed in OpenSSL 1.1.1n (Affected 1.1.1-1.1.1m). Fixed in OpenSSL 1.0.2zd (Affected 1.0.2-1.0.2zc)."

var nvdValue858 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue857}

var nvdValue859 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue858}, ID: "CVE-2022-0778", LastModified: "2026-06-17T04:21:13.530", Metrics: &nvdValue511, Published: "2022-03-15T17:15:08.513"}

var nvdValue860 = "In Spring Cloud Function versions 3.1.6, 3.2.2 and older unsupported versions, when using routing functionality it is possible for a user to provide a specially crafted SpEL as a routing-expression that may result in remote code execution and access to local resources."

var nvdValue861 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue860}

var nvdValue862 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue861}, ID: "CVE-2022-22963", LastModified: "2026-06-17T04:29:15.340", Metrics: &nvdValue30, Published: "2022-04-01T23:15:13.663"}

var nvdValue863 = "A Spring MVC or Spring WebFlux application running on JDK 9+ may be vulnerable to remote code execution (RCE) via data binding. The specific exploit requires the application to run on Tomcat as a WAR deployment. If the application is deployed as a Spring Boot executable jar, i.e. the default, it is not vulnerable to the exploit. However, the nature of the vulnerability is more general, and there may be other ways to exploit it."

var nvdValue864 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue863}

var nvdValue865 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue864}, ID: "CVE-2022-22965", LastModified: "2026-06-17T04:29:15.610", Metrics: &nvdValue30, Published: "2022-04-01T23:15:13.870"}

var nvdValue866 = "An arbitrary file write vulnerability in Express-FileUpload v1.3.1 allows attackers to upload multiple files with the same name, causing an overwrite of files in the web application server."

var nvdValue867 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue866}

var nvdValue868 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue9}, CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue613}}

var nvdValue869 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue867}, ID: "CVE-2022-27261", LastModified: "2026-06-17T04:36:37.007", Metrics: &nvdValue868, Published: "2022-04-12T17:15:09.973"}

var nvdValue870 = "The c_rehash script does not properly sanitise shell metacharacters to prevent command injection. This script is distributed by some operating systems in a manner where it is automatically executed. On such operating systems, an attacker could execute arbitrary commands with the privileges of the script. Use of the c_rehash script is considered obsolete and should be replaced by the OpenSSL rehash command line tool. Fixed in OpenSSL 3.0.3 (Affected 3.0.0,3.0.1,3.0.2). Fixed in OpenSSL 1.1.1o (Affected 1.1.1-1.1.1n). Fixed in OpenSSL 1.0.2ze (Affected 1.0.2-1.0.2zd)."

var nvdValue871 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue870}

var nvdValue872 = schema.CVSSV31{BaseScore: 7.3, VectorString: "CVSS:3.1/AV:L/AC:L/PR:L/UI:R/S:U/C:H/I:H/A:H", Version: "3.1"}

var nvdValue873 = schema.CVEAPIJSON20CVSSV31{CvssData: &nvdValue872}

var nvdValue874 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue59}, CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue873}}

var nvdValue875 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue871}, ID: "CVE-2022-1292", LastModified: "2026-06-17T04:22:09.873", Metrics: &nvdValue874, Published: "2022-05-03T16:15:18.823"}

var nvdValue876 = "The function `OCSP_basic_verify` verifies the signer certificate on an OCSP response. In the case where the (non-default) flag OCSP_NOCHECKS is used then the response will be positive (meaning a successful verification) even in the case where the response signing certificate fails to verify. It is anticipated that most users of `OCSP_basic_verify` will not use the OCSP_NOCHECKS flag. In this case the `OCSP_basic_verify` function will return a negative value (indicating a fatal error) in the case of a certificate verification failure. The normal expected return value in this case would be 0. This issue also impacts the command line OpenSSL \"ocsp\" application. When verifying an ocsp response with the \"-no_cert_checks\" option the command line application will report that the verification is successful even though it has in fact failed. In this case the incorrect successful response will also be accompanied by error messages showing the failure and contradicting the apparently successful result. Fixed in OpenSSL 3.0.3 (Affected 3.0.0,3.0.1,3.0.2)."

var nvdValue877 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue876}

var nvdValue878 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue9}, CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue809}}

var nvdValue879 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue877}, ID: "CVE-2022-1343", LastModified: "2026-06-17T04:22:15.263", Metrics: &nvdValue878, Published: "2022-05-03T16:15:18.873"}

var nvdValue880 = "The OpenSSL 3.0 implementation of the RC4-MD5 ciphersuite incorrectly uses the AAD data as the MAC key. This makes the MAC key trivially predictable. An attacker could exploit this issue by performing a man-in-the-middle attack to modify data being sent from one endpoint to an OpenSSL 3.0 recipient such that the modified data would still pass the MAC integrity check. Note that data sent from an OpenSSL 3.0 endpoint to a non-OpenSSL 3.0 endpoint will always be rejected by the recipient and the connection will fail at that point. Many application protocols require data to be sent from the client to the server first. Therefore, in such a case, only an OpenSSL 3.0 server would be impacted when talking to a non-OpenSSL 3.0 client. If both endpoints are OpenSSL 3.0 then the attacker could modify data being sent in both directions. In this case both clients and servers could be affected, regardless of the application protocol. Note that in the absence of an attacker this bug means that an OpenSSL 3.0 endpoint communicating with a non-OpenSSL 3.0 endpoint will fail to complete the handshake when using this ciphersuite. The confidentiality of data is not impacted by this issue, i.e. an attacker cannot decrypt data that has been encrypted using this ciphersuite - they can only modify it. In order for this attack to work both endpoints must legitimately negotiate the RC4-MD5 ciphersuite. This ciphersuite is not compiled by default in OpenSSL 3.0, and is not available within the default provider or the default ciphersuite list. This ciphersuite will never be used if TLSv1.3 has been negotiated. In order for an OpenSSL 3.0 endpoint to use this ciphersuite the following must have occurred: 1) OpenSSL must have been compiled with the (non-default) compile time option enable-weak-ssl-ciphers 2) OpenSSL must have had the legacy provider explicitly loaded (either through application code or via configuration) 3) The ciphersuite must have been explicitly added to the ciphersuite list 4) The libssl security level must have been set to 0 (default is 1) 5) A version of SSL/TLS below TLSv1.3 must have been negotiated 6) Both endpoints must negotiate the RC4-MD5 ciphersuite in preference to any others that both endpoints have in common Fixed in OpenSSL 3.0.3 (Affected 3.0.0,3.0.1,3.0.2)."

var nvdValue881 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue880}

var nvdValue882 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue881}, ID: "CVE-2022-1434", LastModified: "2026-06-17T04:22:26.790", Metrics: &nvdValue773, Published: "2022-05-03T16:15:18.917"}

var nvdValue883 = "The OPENSSL_LH_flush() function, which empties a hash table, contains a bug that breaks reuse of the memory occuppied by the removed hash table entries. This function is used when decoding certificates or keys. If a long lived process periodically decodes certificates or keys its memory usage will expand without bounds and the process might be terminated by the operating system causing a denial of service. Also traversing the empty hash table entries will take increasingly more time. Typically such long lived processes might be TLS clients or TLS servers configured to accept client certificate authentication. The function was added in the OpenSSL 3.0 version thus older releases are not affected by the issue. Fixed in OpenSSL 3.0.3 (Affected 3.0.0,3.0.1,3.0.2)."

var nvdValue884 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue883}

var nvdValue885 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue884}, ID: "CVE-2022-1473", LastModified: "2026-06-17T04:22:31.053", Metrics: &nvdValue511, Published: "2022-05-03T16:15:18.957"}

var nvdValue886 = "The documentation of Apache Tomcat 10.1.0-M1 to 10.1.0-M14, 10.0.0-M1 to 10.0.20, 9.0.13 to 9.0.62 and 8.5.38 to 8.5.78 for the EncryptInterceptor incorrectly stated it enabled Tomcat clustering to run over an untrusted network. This was not correct. While the EncryptInterceptor does provide confidentiality and integrity protection, it does not protect against all risks associated with running over any untrusted network, particularly DoS risks."

var nvdValue887 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue886}

var nvdValue888 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue887}, ID: "CVE-2022-29885", LastModified: "2026-06-17T04:40:54.260", Metrics: &nvdValue511, Published: "2022-05-12T08:15:07.630"}

var nvdValue889 = "Improper Removal of Sensitive Information Before Storage or Transfer in GitHub repository eventsource/eventsource prior to v2.0.2."

var nvdValue890 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue889}

var nvdValue891 = schema.CVSSV31{BaseScore: 9.3, VectorString: "CVSS:3.1/AV:N/AC:L/PR:N/UI:R/S:C/C:H/I:H/A:N", Version: "3.1"}

var nvdValue892 = schema.CVEAPIJSON20CVSSV31{CvssData: &nvdValue891}

var nvdValue893 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue95}, CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue892}}

var nvdValue894 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue890}, ID: "CVE-2022-1650", LastModified: "2026-06-17T04:22:51.493", Metrics: &nvdValue893, Published: "2022-05-12T11:15:07.290"}

var nvdValue895 = "A use-after-free in Busybox 1.35-x's awk applet leads to denial of service and possibly code execution when processing a crafted awk pattern in the copyvar function."

var nvdValue896 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue895}

var nvdValue897 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue896}, ID: "CVE-2022-30065", LastModified: "2026-07-07T19:16:47.807", Metrics: &nvdValue801, Published: "2022-05-18T15:15:10.240"}

var nvdValue898 = schema.CVEAPIJSON20LangString{Lang: "en", Value: "Out-of-bounds Write in GitHub repository vim/vim prior to 8.2.4977."}

var nvdValue899 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue898}, ID: "CVE-2022-1785", LastModified: "2026-06-17T04:23:06.517", Metrics: &nvdValue450, Published: "2022-05-19T13:15:07.780"}

var nvdValue900 = "In spring security versions prior to 5.4.11+, 5.5.7+ , 5.6.4+ and older unsupported versions, RegexRequestMatcher can easily be misconfigured to be bypassed on some servlet containers. Applications using RegexRequestMatcher with `.` in the regular expression are possibly vulnerable to an authorization bypass."

var nvdValue901 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue900}

var nvdValue902 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue901}, ID: "CVE-2022-22978", LastModified: "2026-06-17T04:29:17.077", Metrics: &nvdValue30, Published: "2022-05-19T15:15:08.057"}

var nvdValue903 = "An improper authentication vulnerability exists in curl 7.33.0 to and including 7.82.0 which might allow reuse OAUTH2-authenticated connections without properly making sure that the connection was authenticated with the same credentials as set for this transfer. This affects SASL-enabled protocols: SMPTP(S), IMAP(S), POP3(S) and LDAP(S) (openldap only)."

var nvdValue904 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue903}

var nvdValue905 = schema.CVSSV20{BaseScore: 5.5, VectorString: "AV:N/AC:L/Au:S/C:P/I:P/A:N", Version: "2.0"}

var nvdValue906 = schema.CVEAPIJSON20CVSSV2{CvssData: &nvdValue905}

var nvdValue907 = schema.CVSSV31{BaseScore: 8.1, VectorString: "CVSS:3.1/AV:N/AC:L/PR:L/UI:N/S:U/C:H/I:H/A:N", Version: "3.1"}

var nvdValue908 = schema.CVEAPIJSON20CVSSV31{CvssData: &nvdValue907}

var nvdValue909 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue906}, CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue908}}

var nvdValue910 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue904}, ID: "CVE-2022-22576", LastModified: "2026-06-17T04:28:37.280", Metrics: &nvdValue909, Published: "2022-05-26T17:15:09.077"}

var nvdValue911 = schema.CVEAPIJSON20LangString{Lang: "en", Value: "Out-of-bounds Write in GitHub repository vim/vim prior to 8.2."}

var nvdValue912 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue911}, ID: "CVE-2022-1897", LastModified: "2026-06-17T04:23:19.087", Metrics: &nvdValue801, Published: "2022-05-27T15:15:07.620"}

var nvdValue913 = schema.CVEAPIJSON20LangString{Lang: "en", Value: "Buffer Over-read in GitHub repository vim/vim prior to 8.2."}

var nvdValue914 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue913}, ID: "CVE-2022-1927", LastModified: "2026-06-17T04:23:22.170", Metrics: &nvdValue801, Published: "2022-05-29T14:15:08.047"}

var nvdValue915 = "An insufficiently protected credentials vulnerability exists in curl 4.9 to and include curl 7.82.0 are affected that could allow an attacker to extract credentials when follows HTTP(S) redirects is used with authentication could leak credentials to other services that exist on different protocols or port numbers."

var nvdValue916 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue915}

var nvdValue917 = schema.CVSSV20{BaseScore: 3.5, VectorString: "AV:N/AC:M/Au:S/C:P/I:N/A:N", Version: "2.0"}

var nvdValue918 = schema.CVEAPIJSON20CVSSV2{CvssData: &nvdValue917}

var nvdValue919 = schema.CVSSV31{BaseScore: 5.7, VectorString: "CVSS:3.1/AV:N/AC:L/PR:L/UI:R/S:U/C:H/I:N/A:N", Version: "3.1"}

var nvdValue920 = schema.CVEAPIJSON20CVSSV31{CvssData: &nvdValue919}

var nvdValue921 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue918}, CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue920}}

var nvdValue922 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue916}, ID: "CVE-2022-27774", LastModified: "2026-06-17T04:37:28.907", Metrics: &nvdValue921, Published: "2022-06-02T14:15:43.317"}

var nvdValue923 = "An information disclosure vulnerability exists in curl 7.65.0 to 7.82.0 are vulnerable that by using an IPv6 address that was in the connection pool but with a different zone id it could reuse a connection instead."

var nvdValue924 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue923}

var nvdValue925 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue924}, ID: "CVE-2022-27775", LastModified: "2026-06-17T04:37:29.077", Metrics: &nvdValue270, Published: "2022-06-02T14:15:43.510"}

var nvdValue926 = "A insufficiently protected credentials vulnerability in fixed in curl 7.83.0 might leak authentication or cookie header data on HTTP redirects to the same host but another port number."

var nvdValue927 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue926}

var nvdValue928 = schema.CVSSV31{BaseScore: 6.5, VectorString: "CVSS:3.1/AV:N/AC:L/PR:N/UI:R/S:U/C:H/I:N/A:N", Version: "3.1"}

var nvdValue929 = schema.CVEAPIJSON20CVSSV31{CvssData: &nvdValue928}

var nvdValue930 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue203}, CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue929}}

var nvdValue931 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue927}, ID: "CVE-2022-27776", LastModified: "2026-06-17T04:37:29.233", Metrics: &nvdValue930, Published: "2022-06-02T14:15:43.713"}

var nvdValue932 = "libcurl provides the `CURLOPT_CERTINFO` option to allow applications torequest details to be returned about a server's certificate chain.Due to an erroneous function, a malicious server could make libcurl built withNSS get stuck in a never-ending busy-loop when trying to retrieve thatinformation."

var nvdValue933 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue932}

var nvdValue934 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue933}, ID: "CVE-2022-27781", LastModified: "2026-06-17T04:37:29.900", Metrics: &nvdValue511, Published: "2022-06-02T14:15:44.467"}

var nvdValue935 = "libcurl would reuse a previously created connection even when a TLS or SSHrelated option had been changed that should have prohibited reuse.libcurl keeps previously used connections in a connection pool for subsequenttransfers to reuse if one of them matches the setup. However, several TLS andSSH settings were left out from the configuration match checks, making themmatch too easily."

var nvdValue936 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue935}

var nvdValue937 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue936}, ID: "CVE-2022-27782", LastModified: "2026-06-17T04:37:30.070", Metrics: &nvdValue614, Published: "2022-06-02T14:15:44.663"}

var nvdValue938 = "In addition to the c_rehash shell command injection identified in CVE-2022-1292, further circumstances where the c_rehash script does not properly sanitise shell metacharacters to prevent command injection were found by code review. When the CVE-2022-1292 was fixed it was not discovered that there are other places in the script where the file names of certificates being hashed were possibly passed to a command executed through the shell. This script is distributed by some operating systems in a manner where it is automatically executed. On such operating systems, an attacker could execute arbitrary commands with the privileges of the script. Use of the c_rehash script is considered obsolete and should be replaced by the OpenSSL rehash command line tool. Fixed in OpenSSL 3.0.4 (Affected 3.0.0,3.0.1,3.0.2,3.0.3). Fixed in OpenSSL 1.1.1p (Affected 1.1.1-1.1.1o). Fixed in OpenSSL 1.0.2zf (Affected 1.0.2-1.0.2ze)."

var nvdValue939 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue938}

var nvdValue940 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue939}, ID: "CVE-2022-2068", LastModified: "2026-06-17T04:41:13.477", Metrics: &nvdValue874, Published: "2022-06-21T15:15:09.060"}

var nvdValue941 = "Jenkins JUnit Plugin 1119.va_a_5e9068da_d7 and earlier does not escape descriptions of test results, resulting in a stored cross-site scripting (XSS) vulnerability exploitable by attackers with Run/Update permission."

var nvdValue942 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue941}

var nvdValue943 = schema.CVSSV20{BaseScore: 3.5, VectorString: "AV:N/AC:M/Au:S/C:N/I:P/A:N", Version: "2.0"}

var nvdValue944 = schema.CVEAPIJSON20CVSSV2{CvssData: &nvdValue943}

var nvdValue945 = schema.CVSSV31{BaseScore: 5.4, VectorString: "CVSS:3.1/AV:N/AC:L/PR:L/UI:R/S:C/C:L/I:L/A:N", Version: "3.1"}

var nvdValue946 = schema.CVEAPIJSON20CVSSV31{CvssData: &nvdValue945}

var nvdValue947 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue944}, CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue946}}

var nvdValue948 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue942}, ID: "CVE-2022-34176", LastModified: "2026-06-17T04:49:50.983", Metrics: &nvdValue947, Published: "2022-06-23T17:15:15.620"}

var nvdValue949 = "Jenkins Pipeline: Input Step Plugin 448.v37cea_9a_10a_70 and earlier archives files uploaded for `file` parameters for Pipeline `input` steps on the controller as part of build metadata, using the parameter name without sanitization as a relative path inside a build-related directory, allowing attackers able to configure Pipelines to create or replace arbitrary files on the Jenkins controller file system with attacker-specified content."

var nvdValue950 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue949}

var nvdValue951 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue950}, ID: "CVE-2022-34177", LastModified: "2026-06-17T04:49:51.080", Metrics: &nvdValue614, Published: "2022-06-23T17:15:15.680"}

var nvdValue952 = "AES OCB mode for 32-bit x86 platforms using the AES-NI assembly optimised implementation will not encrypt the entirety of the data under some circumstances. This could reveal sixteen bytes of data that was preexisting in the memory that wasn't written. In the special case of \"in place\" encryption, sixteen bytes of the plaintext would be revealed. Since OpenSSL does not support OCB based cipher suites for TLS and DTLS, they are both unaffected. Fixed in OpenSSL 3.0.5 (Affected 3.0.0-3.0.4). Fixed in OpenSSL 1.1.1q (Affected 1.1.1-1.1.1p)."

var nvdValue953 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue952}

var nvdValue954 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue953}, ID: "CVE-2022-2097", LastModified: "2026-06-17T04:41:16.520", Metrics: &nvdValue459, Published: "2022-07-05T11:15:08.340"}

var nvdValue955 = "curl < 7.84.0 supports \"chained\" HTTP compression algorithms, meaning that a serverresponse can be compressed multiple times and potentially with different algorithms. The number of acceptable \"links\" in this \"decompression chain\" was unbounded, allowing a malicious server to insert a virtually unlimited number of compression steps.The use of such a decompression chain could result in a \"malloc bomb\", makingcurl end up spending enormous amounts of allocated heap memory, or trying toand returning out of memory errors."

var nvdValue956 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue955}

var nvdValue957 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue956}, ID: "CVE-2022-32206", LastModified: "2026-06-17T04:46:51.593", Metrics: &nvdValue354, Published: "2022-07-07T13:15:08.340"}

var nvdValue958 = "When curl < 7.84.0 does FTP transfers secured by krb5, it handles message verification failures wrongly. This flaw makes it possible for a Man-In-The-Middle attack to go unnoticed and even allows it to inject data to the client."

var nvdValue959 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue958}

var nvdValue960 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue959}, ID: "CVE-2022-32208", LastModified: "2026-06-17T04:46:52.120", Metrics: &nvdValue369, Published: "2022-07-07T13:15:08.467"}

var nvdValue961 = "A OS Command Injection vulnerability exists in Node.js versions <14.20.0, <16.20.0, <18.5.0 due to an insufficient IsAllowedHost check that can easily be bypassed because IsIPAddress does not properly check if an IP address is invalid before making DBS requests allowing rebinding attacks."

var nvdValue962 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue961}

var nvdValue963 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue83}}

var nvdValue964 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue962}, ID: "CVE-2022-32212", LastModified: "2026-06-17T04:46:52.940", Metrics: &nvdValue963, Published: "2022-07-14T15:15:08.237"}

var nvdValue965 = "The llhttp parser <v14.20.1, <v16.17.1 and <v18.9.1 in the http module in Node.js does not correctly parse and validate Transfer-Encoding headers and can lead to HTTP Request Smuggling (HRS)."

var nvdValue966 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue965}

var nvdValue967 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue659}}

var nvdValue968 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue966}, ID: "CVE-2022-32213", LastModified: "2026-06-17T04:46:53.047", Metrics: &nvdValue967, Published: "2022-07-14T15:15:08.287"}

var nvdValue969 = "The llhttp parser <v14.20.1, <v16.17.1 and <v18.9.1 in the http module in Node.js does not strictly use the CRLF sequence to delimit HTTP requests. This can lead to HTTP Request Smuggling (HRS)."

var nvdValue970 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue969}

var nvdValue971 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue970}, ID: "CVE-2022-32214", LastModified: "2026-06-17T04:46:53.167", Metrics: &nvdValue967, Published: "2022-07-14T15:15:08.337"}

var nvdValue972 = "The llhttp parser <v14.20.1, <v16.17.1 and <v18.9.1 in the http module in Node.js does not correctly handle multi-line Transfer-Encoding headers. This can lead to HTTP Request Smuggling (HRS)."

var nvdValue973 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue972}

var nvdValue974 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue973}, ID: "CVE-2022-32215", LastModified: "2026-06-17T04:46:53.277", Metrics: &nvdValue967, Published: "2022-07-14T15:15:08.387"}

var nvdValue975 = "A cryptographic vulnerability exists on Node.js on linux in versions of 18.x prior to 18.40.0 which allowed a default path for openssl.cnf that might be accessible under some circumstances to a non-admin user instead of /etc/ssl as was the case in versions prior to the upgrade to OpenSSL 3."

var nvdValue976 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue975}

var nvdValue977 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue809}}

var nvdValue978 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue976}, ID: "CVE-2022-32222", LastModified: "2026-06-17T04:46:54.617", Metrics: &nvdValue977, Published: "2022-07-14T15:15:08.437"}

var nvdValue979 = "Node.js is vulnerable to Hijack Execution Flow: DLL Hijacking under certain conditions on Windows platforms.This vulnerability can be exploited if the victim has the following dependencies on a Windows machine:* OpenSSL has been installed and “C:\\Program Files\\Common Files\\SSL\\openssl.cnf” exists.Whenever the above conditions are present, `node.exe` will search for `providers.dll` in the current user directory.After that, `node.exe` will try to search for `providers.dll` by the DLL Search Order in Windows.It is possible for an attacker to place the malicious file `providers.dll` under a variety of paths and exploit this vulnerability."

var nvdValue980 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue979}

var nvdValue981 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue873}}

var nvdValue982 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue980}, ID: "CVE-2022-32223", LastModified: "2026-06-17T04:46:54.723", Metrics: &nvdValue981, Published: "2022-07-14T15:15:08.487"}

var nvdValue983 = schema.CVEAPIJSON20LangString{Lang: "en", Value: "libexpat before 2.4.9 has a use-after-free in the doContent function in xmlparse.c."}

var nvdValue984 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue983}, ID: "CVE-2022-40674", LastModified: "2026-06-17T05:01:48.967", Metrics: &nvdValue963, Published: "2022-09-14T11:15:54.020"}

var nvdValue985 = "When curl is used to retrieve and parse cookies from a HTTP(S) server, itaccepts cookies using control codes that when later are sent back to a HTTPserver might make the server return 400 responses. Effectively allowing a\"sister site\" to deny service to all siblings."

var nvdValue986 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue985}

var nvdValue987 = schema.CVSSV31{BaseScore: 3.7, VectorString: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:N/I:N/A:L", Version: "3.1"}

var nvdValue988 = schema.CVEAPIJSON20CVSSV31{CvssData: &nvdValue987}

var nvdValue989 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue988}}

var nvdValue990 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue986}, ID: "CVE-2022-35252", LastModified: "2026-06-17T04:51:39.513", Metrics: &nvdValue989, Published: "2022-09-23T14:15:12.323"}

var nvdValue991 = "OpenSSL supports creating a custom cipher via the legacy EVP_CIPHER_meth_new() function and associated function calls. This function was deprecated in OpenSSL 3.0 and application authors are instead encouraged to use the new provider mechanism in order to implement custom ciphers. OpenSSL versions 3.0.0 to 3.0.5 incorrectly handle legacy custom ciphers passed to the EVP_EncryptInit_ex2(), EVP_DecryptInit_ex2() and EVP_CipherInit_ex2() functions (as well as other similarly named encryption and decryption initialisation functions). Instead of using the custom cipher directly it incorrectly tries to fetch an equivalent cipher from the available providers. An equivalent cipher is found based on the NID passed to EVP_CIPHER_meth_new(). This NID is supposed to represent the unique NID for a given cipher. However it is possible for an application to incorrectly pass NID_undef as this value in the call to EVP_CIPHER_meth_new(). When NID_undef is used in this way the OpenSSL encryption/decryption initialisation function will match the NULL cipher as being equivalent and will fetch this from the available providers. This will succeed if the default provider has been loaded (or if a third party provider has been loaded that offers this cipher). Using the NULL cipher means that the plaintext is emitted as the ciphertext. Applications are only affected by this issue if they call EVP_CIPHER_meth_new() using NID_undef and subsequently use it in a call to an encryption/decryption initialisation function. Applications that only use SSL/TLS are not impacted by this issue. Fixed in OpenSSL 3.0.6 (Affected 3.0.0-3.0.5)."

var nvdValue992 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue991}

var nvdValue993 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue269}}

var nvdValue994 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue992}, ID: "CVE-2022-3358", LastModified: "2026-06-17T04:59:22.627", Metrics: &nvdValue993, Published: "2022-10-11T15:15:10.233"}

var nvdValue995 = "In libexpat through 2.4.9, there is a use-after free caused by overeager destruction of a shared DTD in XML_ExternalEntityParserCreate in out-of-memory situations."

var nvdValue996 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue995}

var nvdValue997 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue377}}

var nvdValue998 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue996}, ID: "CVE-2022-43680", LastModified: "2026-06-17T05:07:06.373", Metrics: &nvdValue997, Published: "2022-10-24T14:15:53.323"}

var nvdValue999 = "A buffer overrun can be triggered in X.509 certificate verification, specifically in name constraint checking. Note that this occurs after certificate chain signature verification and requires either a CA to have signed the malicious certificate or for the application to continue certificate verification despite failure to construct a path to a trusted issuer. An attacker can craft a malicious email address to overflow four attacker-controlled bytes on the stack. This buffer overflow could result in a crash (causing a denial of service) or potentially remote code execution. Many platforms implement stack overflow protections which would mitigate against the risk of remote code execution. The risk may be further mitigated based on stack layout for any given platform/compiler. Pre-announcements of CVE-2022-3602 described this issue as CRITICAL. Further analysis based on some of the mitigating factors described above have led this to be downgraded to HIGH. Users are still encouraged to upgrade to a new version as soon as possible. In a TLS client, this can be triggered by connecting to a malicious server. In a TLS server, this can be triggered if the server requests client authentication and a malicious client connects. Fixed in OpenSSL 3.0.7 (Affected 3.0.0,3.0.1,3.0.2,3.0.3,3.0.4,3.0.5,3.0.6)."

var nvdValue1000 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue999}

var nvdValue1001 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1000}, ID: "CVE-2022-3602", LastModified: "2026-06-17T04:59:49.620", Metrics: &nvdValue997, Published: "2022-11-01T18:15:10.983"}

var nvdValue1002 = "A buffer overrun can be triggered in X.509 certificate verification, specifically in name constraint checking. Note that this occurs after certificate chain signature verification and requires either a CA to have signed a malicious certificate or for an application to continue certificate verification despite failure to construct a path to a trusted issuer. An attacker can craft a malicious email address in a certificate to overflow an arbitrary number of bytes containing the `.' character (decimal 46) on the stack. This buffer overflow could result in a crash (causing a denial of service). In a TLS client, this can be triggered by connecting to a malicious server. In a TLS server, this can be triggered if the server requests client authentication and a malicious client connects.\n\n"

var nvdValue1003 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1002}

var nvdValue1004 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1003}, ID: "CVE-2022-3786", LastModified: "2026-06-17T05:00:18.193", Metrics: &nvdValue997, Published: "2022-11-01T18:15:11.047"}

var nvdValue1005 = "When doing HTTP(S) transfers, libcurl might erroneously use the read callback (`CURLOPT_READFUNCTION`) to ask for data to send, even when the `CURLOPT_POSTFIELDS` option has been set, if the same handle previously was used to issue a `PUT` request which used that callback. This flaw may surprise the application and cause it to misbehave and either send off the wrong data or use memory after free or similar in the subsequent `POST` request. The problem exists in the logic for a reused handle when it is changed from a PUT to a POST."

var nvdValue1006 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1005}

var nvdValue1007 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue29}}

var nvdValue1008 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1006}, ID: "CVE-2022-32221", LastModified: "2026-06-17T04:46:54.347", Metrics: &nvdValue1007, Published: "2022-12-05T22:15:10.343"}

var nvdValue1009 = "A weak randomness in WebCrypto keygen vulnerability exists in Node.js 18 due to a change with EntropySource() in SecretKeyGenTraits::DoKeyGen() in src/crypto/crypto_keygen.cc. There are two problems with this: 1) It does not check the return value, it assumes EntropySource() always succeeds, but it can (and sometimes will) fail. 2) The random data returned byEntropySource() may not be cryptographically strong and therefore not suitable as keying material."

var nvdValue1010 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1009}

var nvdValue1011 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue700}}

var nvdValue1012 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1010}, ID: "CVE-2022-35255", LastModified: "2026-06-17T04:51:39.850", Metrics: &nvdValue1011, Published: "2022-12-05T22:15:10.513"}

var nvdValue1013 = "The llhttp parser in the http module in Node v18.7.0 does not correctly handle header fields that are not terminated with CLRF. This may result in HTTP Request Smuggling."

var nvdValue1014 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1013}

var nvdValue1015 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1014}, ID: "CVE-2022-35256", LastModified: "2026-06-17T04:51:40.007", Metrics: &nvdValue967, Published: "2022-12-05T22:15:10.570"}

var nvdValue1016 = "A OS Command Injection vulnerability exists in Node.js versions <14.21.1, <16.18.1, <18.12.1, <19.0.1 due to an insufficient IsAllowedHost check that can easily be bypassed because IsIPAddress does not properly check if an IP address is invalid before making DBS requests allowing rebinding attacks.The fix for this issue in https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2022-32212 was incomplete and this new CVE is to complete the fix."

var nvdValue1017 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1016}

var nvdValue1018 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1017}, ID: "CVE-2022-43548", LastModified: "2026-06-17T05:06:48.960", Metrics: &nvdValue963, Published: "2022-12-05T22:15:10.923"}

var nvdValue1019 = "A timing based side channel exists in the OpenSSL RSA Decryption implementation\nwhich could be sufficient to recover a plaintext across a network in a\nBleichenbacher style attack. To achieve a successful decryption an attacker\nwould have to be able to send a very large number of trial messages for\ndecryption. The vulnerability affects all RSA padding modes: PKCS#1 v1.5,\nRSA-OEAP and RSASVE.\n\nFor example, in a TLS connection, RSA is commonly used by a client to send an\nencrypted pre-master secret to the server. An attacker that had observed a\ngenuine connection between a client and a server could use this flaw to send\ntrial messages to the server and record the time taken to process them. After a\nsufficiently large number of messages the attacker could recover the pre-master\nsecret used for the original connection and thus be able to decrypt the\napplication data sent over that connection."

var nvdValue1020 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1019}

var nvdValue1021 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue368}}

var nvdValue1022 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1020}, ID: "CVE-2022-4304", LastModified: "2026-06-17T05:20:32.457", Metrics: &nvdValue1021, Published: "2023-02-08T20:15:23.887"}

var nvdValue1023 = "The function PEM_read_bio_ex() reads a PEM file from a BIO and parses and\ndecodes the \"name\" (e.g. \"CERTIFICATE\"), any header data and the payload data.\nIf the function succeeds then the \"name_out\", \"header\" and \"data\" arguments are\npopulated with pointers to buffers containing the relevant decoded data. The\ncaller is responsible for freeing those buffers. It is possible to construct a\nPEM file that results in 0 bytes of payload data. In this case PEM_read_bio_ex()\nwill return a failure code but will populate the header argument with a pointer\nto a buffer that has already been freed. If the caller also frees this buffer\nthen a double free will occur. This will most likely lead to a crash. This\ncould be exploited by an attacker who has the ability to supply malicious PEM\nfiles for parsing to achieve a denial of service attack.\n\nThe functions PEM_read_bio() and PEM_read() are simple wrappers around\nPEM_read_bio_ex() and therefore these functions are also directly affected.\n\nThese functions are also called indirectly by a number of other OpenSSL\nfunctions including PEM_X509_INFO_read_bio_ex() and\nSSL_CTX_use_serverinfo_file() which are also vulnerable. Some OpenSSL internal\nuses of these functions are not vulnerable because the caller does not free the\nheader argument if PEM_read_bio_ex() returns a failure code. These locations\ninclude the PEM_read_bio_TYPE() functions as well as the decoders introduced in\nOpenSSL 3.0.\n\nThe OpenSSL asn1parse command line application is also impacted by this issue."

var nvdValue1024 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1023}

var nvdValue1025 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1024}, ID: "CVE-2022-4450", LastModified: "2026-06-17T05:20:54.027", Metrics: &nvdValue997, Published: "2023-02-08T20:15:23.973"}

var nvdValue1026 = "A use after free vulnerability exists in curl <7.87.0. Curl can be asked to *tunnel* virtually all protocols it supports through an HTTP proxy. HTTP proxies can (and often do) deny such tunnel operations. When getting denied to tunnel the specific protocols SMB or TELNET, curl would use a heap-allocated struct after it had been freed, in its transfer shutdown code path."

var nvdValue1027 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1026}

var nvdValue1028 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue619}}

var nvdValue1029 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1027}, ID: "CVE-2022-43552", LastModified: "2026-06-17T05:06:49.660", Metrics: &nvdValue1028, Published: "2023-02-09T20:15:10.950"}

var nvdValue1030 = "GnuPG can be made to spin on a relatively small input by (for example) crafting a public key with thousands of signatures attached, compressed down to just a few KB."

var nvdValue1031 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1030}

var nvdValue1032 = schema.CVSSV31{BaseScore: 3.3, VectorString: "CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:N/I:N/A:L", Version: "3.1"}

var nvdValue1033 = schema.CVEAPIJSON20CVSSV31{CvssData: &nvdValue1032}

var nvdValue1034 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue1033}}

var nvdValue1035 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1031}, ID: "CVE-2022-3219", LastModified: "2026-06-17T04:59:05.937", Metrics: &nvdValue1034, Published: "2023-02-23T20:15:12.393"}

var nvdValue1036 = "Versions of the package semver before 7.5.2 are vulnerable to Regular Expression Denial of Service (ReDoS) via the function new Range, when untrusted user data is provided as a range.\r\r\r"

var nvdValue1037 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1036}

var nvdValue1038 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1037}, ID: "CVE-2022-25883", LastModified: "2026-06-17T04:34:26.360", Metrics: &nvdValue997, Published: "2023-06-21T05:15:09.060"}

var nvdValue1039 = "A vulnerability was found in GNU C Library 2.38. It has been declared as critical. This vulnerability affects the function __monstartup of the file gmon.c of the component Call Graph Monitor. The manipulation leads to buffer overflow. It is recommended to apply a patch to fix this issue. VDB-220246 is the identifier assigned to this vulnerability. NOTE: The real existence of this vulnerability is still doubted at the moment. The inputs that induce this vulnerability are basically addresses of the running application that is built with gmon enabled. It's basically trusted input or input that needs an actual security flaw to be compromised or controlled."

var nvdValue1040 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1039}

var nvdValue1041 = schema.CVSSV20{BaseScore: 4, VectorString: "AV:A/AC:H/Au:S/C:P/I:P/A:P", Version: "2.0"}

var nvdValue1042 = schema.CVEAPIJSON20CVSSV2{CvssData: &nvdValue1041}

var nvdValue1043 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV2: []*schema.CVEAPIJSON20CVSSV2{&nvdValue1042}, CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue29}}

var nvdValue1044 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1040}, ID: "CVE-2023-0687", LastModified: "2026-06-17T05:26:05.437", Metrics: &nvdValue1043, Published: "2023-02-06T19:15:10.260"}

var nvdValue1045 = "The public API function BIO_new_NDEF is a helper function used for streaming\nASN.1 data via a BIO. It is primarily used internally to OpenSSL to support the\nSMIME, CMS and PKCS7 streaming capabilities, but may also be called directly by\nend user applications.\n\nThe function receives a BIO from the caller, prepends a new BIO_f_asn1 filter\nBIO onto the front of it to form a BIO chain, and then returns the new head of\nthe BIO chain to the caller. Under certain conditions, for example if a CMS\nrecipient public key is invalid, the new filter BIO is freed and the function\nreturns a NULL result indicating a failure. However, in this case, the BIO chain\nis not properly cleaned up and the BIO passed by the caller still retains\ninternal pointers to the previously freed filter BIO. If the caller then goes on\nto call BIO_pop() on the BIO then a use-after-free will occur. This will most\nlikely result in a crash.\n\n\n\nThis scenario occurs directly in the internal function B64_write_ASN1() which\nmay cause BIO_new_NDEF() to be called and will subsequently call BIO_pop() on\nthe BIO. This internal function is in turn called by the public API functions\nPEM_write_bio_ASN1_stream, PEM_write_bio_CMS_stream, PEM_write_bio_PKCS7_stream,\nSMIME_write_ASN1, SMIME_write_CMS and SMIME_write_PKCS7.\n\nOther public API functions that may be impacted by this include\ni2d_ASN1_bio_stream, BIO_new_CMS, BIO_new_PKCS7, i2d_CMS_bio_stream and\ni2d_PKCS7_bio_stream.\n\nThe OpenSSL cms and smime command line applications are similarly affected."

var nvdValue1046 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1045}

var nvdValue1047 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1046}, ID: "CVE-2023-0215", LastModified: "2026-06-17T05:25:02.560", Metrics: &nvdValue997, Published: "2023-02-08T20:15:24.107"}

var nvdValue1048 = "There is a type confusion vulnerability relating to X.400 address processing\ninside an X.509 GeneralName. X.400 addresses were parsed as an ASN1_STRING but\nthe public structure definition for GENERAL_NAME incorrectly specified the type\nof the x400Address field as ASN1_TYPE. This field is subsequently interpreted by\nthe OpenSSL function GENERAL_NAME_cmp as an ASN1_TYPE rather than an\nASN1_STRING.\n\nWhen CRL checking is enabled (i.e. the application sets the\nX509_V_FLAG_CRL_CHECK flag), this vulnerability may allow an attacker to pass\narbitrary pointers to a memcmp call, enabling them to read memory contents or\nenact a denial of service. In most cases, the attack requires the attacker to\nprovide both the certificate chain and CRL, neither of which need to have a\nvalid signature. If the attacker only controls one of these inputs, the other\ninput must already contain an X.400 address as a CRL distribution point, which\nis uncommon. As such, this vulnerability is most likely to only affect\napplications which have implemented their own functionality for retrieving CRLs\nover a network."

var nvdValue1049 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1048}

var nvdValue1050 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue747}}

var nvdValue1051 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1049}, ID: "CVE-2023-0286", LastModified: "2026-06-17T05:25:12.637", Metrics: &nvdValue1050, Published: "2023-02-08T20:15:24.267"}

var nvdValue1052 = "Undici is an HTTP/1.1 client for Node.js. Starting with version 2.0.0 and prior to version 5.19.1, the undici library does not protect `host` HTTP header from CRLF injection vulnerabilities. This issue is patched in Undici v5.19.1. As a workaround, sanitize the `headers.host` string before passing to undici."

var nvdValue1053 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1052}

var nvdValue1054 = schema.CVSSV31{BaseScore: 5.4, VectorString: "CVSS:3.1/AV:N/AC:L/PR:N/UI:R/S:U/C:L/I:L/A:N", Version: "3.1"}

var nvdValue1055 = schema.CVEAPIJSON20CVSSV31{CvssData: &nvdValue1054}

var nvdValue1056 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue1055}}

var nvdValue1057 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1053}, ID: "CVE-2023-23936", LastModified: "2026-06-17T05:38:19.600", Metrics: &nvdValue1056, Published: "2023-02-16T18:15:10.877"}

var nvdValue1058 = "An allocation of resources without limits or throttling vulnerability exists in curl <v7.88.0 based on the \"chained\" HTTP compression algorithms, meaning that a server response can be compressed multiple times and potentially with differentalgorithms. The number of acceptable \"links\" in this \"decompression chain\" wascapped, but the cap was implemented on a per-header basis allowing a maliciousserver to insert a virtually unlimited number of compression steps simply byusing many headers. The use of such a decompression chain could result in a \"malloc bomb\", making curl end up spending enormous amounts of allocated heap memory, or trying to and returning out of memory errors."

var nvdValue1059 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1058}

var nvdValue1060 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue353}}

var nvdValue1061 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1059}, ID: "CVE-2023-23916", LastModified: "2026-06-17T05:38:16.293", Metrics: &nvdValue1060, Published: "2023-02-23T20:15:13.777"}

var nvdValue1062 = "A privilege escalation vulnerability exists in Node.js <19.6.1, <18.14.1, <16.19.1 and <14.21.3 that made it possible to bypass the experimental Permissions (https://nodejs.org/api/permissions.html) feature in Node.js and access non authorized modules by using process.mainModule.require(). This only affects users who had enabled the experimental permissions option with --experimental-policy."

var nvdValue1063 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1062}

var nvdValue1064 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1063}, ID: "CVE-2023-23918", LastModified: "2026-06-17T05:38:16.663", Metrics: &nvdValue993, Published: "2023-02-23T20:15:13.920"}

var nvdValue1065 = "A cryptographic vulnerability exists in Node.js <19.2.0, <18.14.1, <16.19.1, <14.21.3 that in some cases did does not clear the OpenSSL error stack after operations that may set it. This may lead to false positive errors during subsequent cryptographic operations that happen to be on the same thread. This in turn could be used to cause a denial of service."

var nvdValue1066 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1065}

var nvdValue1067 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1066}, ID: "CVE-2023-23919", LastModified: "2026-06-17T05:38:16.867", Metrics: &nvdValue997, Published: "2023-02-23T20:15:13.977"}

var nvdValue1068 = "An untrusted search path vulnerability exists in Node.js. <19.6.1, <18.14.1, <16.19.1, and <14.21.3 that could allow an attacker to search and potentially load ICU data when running with elevated privileges."

var nvdValue1069 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1068}

var nvdValue1070 = schema.CVSSV31{BaseScore: 4.2, VectorString: "CVSS:3.1/AV:L/AC:L/PR:H/UI:R/S:U/C:N/I:H/A:N", Version: "3.1"}

var nvdValue1071 = schema.CVEAPIJSON20CVSSV31{CvssData: &nvdValue1070}

var nvdValue1072 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue1071}}

var nvdValue1073 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1069}, ID: "CVE-2023-23920", LastModified: "2026-06-17T05:38:17.210", Metrics: &nvdValue1072, Published: "2023-02-23T20:15:14.047"}

var nvdValue1074 = "When using the RemoteIpFilter with requests received from a    reverse proxy via HTTP that include the X-Forwarded-Proto    header set to https, session cookies created by Apache Tomcat 11.0.0-M1 to 11.0.0.-M2, 10.1.0-M1 to 10.1.5, 9.0.0-M1 to 9.0.71 and 8.5.0 to 8.5.85 did not\u00a0include the secure attribute. This could result in the user agent\u00a0transmitting the session cookie over an insecure channel.\n\nOlder, EOL versions may also be affected."

var nvdValue1075 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1074}

var nvdValue1076 = schema.CVSSV31{BaseScore: 4.3, VectorString: "CVSS:3.1/AV:N/AC:L/PR:N/UI:R/S:U/C:L/I:N/A:N", Version: "3.1"}

var nvdValue1077 = schema.CVEAPIJSON20CVSSV31{CvssData: &nvdValue1076}

var nvdValue1078 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue1077}}

var nvdValue1079 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1075}, ID: "CVE-2023-28708", LastModified: "2026-06-17T05:48:36.390", Metrics: &nvdValue1078, Published: "2023-03-22T11:15:10.623"}

var nvdValue1080 = "A security vulnerability has been identified in all supported versions\n\nof OpenSSL related to the verification of X.509 certificate chains\nthat include policy constraints.  Attackers may be able to exploit this\nvulnerability by creating a malicious certificate chain that triggers\nexponential use of computational resources, leading to a denial-of-service\n(DoS) attack on affected systems.\n\nPolicy processing is disabled by default but can be enabled by passing\nthe `-policy' argument to the command line utilities or by calling the\n`X509_VERIFY_PARAM_set1_policies()' function."

var nvdValue1081 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1080}

var nvdValue1082 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1081}, ID: "CVE-2023-0464", LastModified: "2026-06-17T05:25:36.637", Metrics: &nvdValue997, Published: "2023-03-22T17:15:13.130"}

var nvdValue1083 = "Applications that use a non-default option when verifying certificates may be\nvulnerable to an attack from a malicious CA to circumvent certain checks.\n\nInvalid certificate policies in leaf certificates are silently ignored by\nOpenSSL and other certificate policy checks are skipped for that certificate.\nA malicious CA could use this to deliberately assert invalid certificate policies\nin order to circumvent policy checking on the certificate altogether.\n\nPolicy processing is disabled by default but can be enabled by passing\nthe `-policy' argument to the command line utilities or by calling the\n`X509_VERIFY_PARAM_set1_policies()' function."

var nvdValue1084 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1083}

var nvdValue1085 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1084}, ID: "CVE-2023-0465", LastModified: "2026-06-17T05:25:36.823", Metrics: &nvdValue977, Published: "2023-03-28T15:15:06.820"}

var nvdValue1086 = "The function X509_VERIFY_PARAM_add0_policy() is documented to\nimplicitly enable the certificate policy check when doing certificate\nverification. However the implementation of the function does not\nenable the check which allows certificates with invalid or incorrect\npolicies to pass the certificate verification.\n\nAs suddenly enabling the policy check could break existing deployments it was\ndecided to keep the existing behavior of the X509_VERIFY_PARAM_add0_policy()\nfunction.\n\nInstead the applications that require OpenSSL to perform certificate\npolicy check need to use X509_VERIFY_PARAM_set1_policies() or explicitly\nenable the policy check by calling X509_VERIFY_PARAM_set_flags() with\nthe X509_V_FLAG_POLICY_CHECK flag argument.\n\nCertificate policy checks are disabled by default in OpenSSL and are not\ncommonly used by applications."

var nvdValue1087 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1086}

var nvdValue1088 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1087}, ID: "CVE-2023-0466", LastModified: "2026-06-17T05:25:37.007", Metrics: &nvdValue977, Published: "2023-03-28T15:15:06.880"}

var nvdValue1089 = "A vulnerability in input validation exists in curl <8.0 during communication using the TELNET protocol may allow an attacker to pass on maliciously crafted user name and \"telnet options\" during server negotiation. The lack of proper input scrubbing allows an attacker to send content or perform option negotiation without the application's intent. This vulnerability could be exploited if an application allows user input, thereby enabling attackers to execute arbitrary code on the system."

var nvdValue1090 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1089}

var nvdValue1091 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue246}}

var nvdValue1092 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1090}, ID: "CVE-2023-27533", LastModified: "2026-06-17T05:45:25.147", Metrics: &nvdValue1091, Published: "2023-03-30T20:15:07.373"}

var nvdValue1093 = "A path traversal vulnerability exists in curl <8.0.0 SFTP implementation causes the tilde (~) character to be wrongly replaced when used as a prefix in the first path element, in addition to its intended use as the first element to indicate a path relative to the user's home directory. Attackers can exploit this flaw to bypass filtering or execute arbitrary code by crafting a path like /~2/foo while accessing a server with a specific user."

var nvdValue1094 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1093}

var nvdValue1095 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue70}}

var nvdValue1096 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1094}, ID: "CVE-2023-27534", LastModified: "2026-06-17T05:45:25.323", Metrics: &nvdValue1095, Published: "2023-03-30T20:15:07.427"}

var nvdValue1097 = "An authentication bypass vulnerability exists in libcurl <8.0.0 in the FTP connection reuse feature that can result in wrong credentials being used during subsequent transfers. Previously created connections are kept in a connection pool for reuse if they match the current setup. However, certain FTP settings such as CURLOPT_FTP_ACCOUNT, CURLOPT_FTP_ALTERNATIVE_TO_USER, CURLOPT_FTP_SSL_CCC, and CURLOPT_USE_SSL were not included in the configuration match checks, causing them to match too easily. This could lead to libcurl using the wrong credentials when performing a transfer, potentially allowing unauthorized access to sensitive information."

var nvdValue1098 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1097}

var nvdValue1099 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1098}, ID: "CVE-2023-27535", LastModified: "2026-06-17T05:45:25.500", Metrics: &nvdValue1021, Published: "2023-03-30T20:15:07.483"}

var nvdValue1100 = "An authentication bypass vulnerability exists libcurl <8.0.0 in the connection reuse feature which can reuse previously established connections with incorrect user permissions due to a failure to check for changes in the CURLOPT_GSSAPI_DELEGATION option. This vulnerability affects krb5/kerberos/negotiate/GSSAPI transfers and could potentially result in unauthorized access to sensitive information. The safest option is to not reuse connections if the CURLOPT_GSSAPI_DELEGATION option has been changed."

var nvdValue1101 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1100}

var nvdValue1102 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1101}, ID: "CVE-2023-27536", LastModified: "2026-06-17T05:45:25.680", Metrics: &nvdValue1021, Published: "2023-03-30T20:15:07.547"}

var nvdValue1103 = "An authentication bypass vulnerability exists in libcurl prior to v8.0.0 where it reuses a previously established SSH connection despite the fact that an SSH option was modified, which should have prevented reuse. libcurl maintains a pool of previously used connections to reuse them for subsequent transfers if the configurations match. However, two SSH settings were omitted from the configuration check, allowing them to match easily, potentially leading to the reuse of an inappropriate connection."

var nvdValue1104 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1103}

var nvdValue1105 = schema.CVSSV31{BaseScore: 5.5, VectorString: "CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:H/I:N/A:N", Version: "3.1"}

var nvdValue1106 = schema.CVEAPIJSON20CVSSV31{CvssData: &nvdValue1105}

var nvdValue1107 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue1106}}

var nvdValue1108 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1104}, ID: "CVE-2023-27538", LastModified: "2026-06-17T05:45:25.983", Metrics: &nvdValue1107, Published: "2023-03-30T20:15:07.677"}

var nvdValue1109 = "An improper certificate validation vulnerability exists in curl <v8.1.0 in the way it supports matching of wildcard patterns when listed as \"Subject Alternative Name\" in TLS server certificates. curl can be built to use its own name matching function for TLS rather than one provided by a TLS library. This private wildcard matching function would match IDN (International Domain Name) hosts incorrectly and could as a result accept patterns that otherwise should mismatch. IDN hostnames are converted to puny code before used for certificate checks. Puny coded names always start with `xn--` and should not be allowed to pattern match, but the wildcard check in curl could still check for `x*`, which would match even though the IDN name most likely contained nothing even resembling an `x`."

var nvdValue1110 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1109}

var nvdValue1111 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue772}}

var nvdValue1112 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1110}, ID: "CVE-2023-28321", LastModified: "2026-06-17T05:47:25.577", Metrics: &nvdValue1111, Published: "2023-05-26T21:15:16.020"}

var nvdValue1113 = "An information disclosure vulnerability exists in curl <v8.1.0 when doing HTTP(S) transfers, libcurl might erroneously use the read callback (`CURLOPT_READFUNCTION`) to ask for data to send, even when the `CURLOPT_POSTFIELDS` option has been set, if the same handle previously wasused to issue a `PUT` request which used that callback. This flaw may surprise the application and cause it to misbehave and either send off the wrong data or use memory after free or similar in the second transfer. The problem exists in the logic for a reused handle when it is (expected to be) changed from a PUT to a POST."

var nvdValue1114 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1113}

var nvdValue1115 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue543}}

var nvdValue1116 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1114}, ID: "CVE-2023-28322", LastModified: "2026-06-17T05:47:25.773", Metrics: &nvdValue1115, Published: "2023-05-26T21:15:16.153"}

var nvdValue1117 = "Issue summary: Processing some specially crafted ASN.1 object identifiers or\ndata containing them may be very slow.\n\nImpact summary: Applications that use OBJ_obj2txt() directly, or use any of\nthe OpenSSL subsystems OCSP, PKCS7/SMIME, CMS, CMP/CRMF or TS with no message\nsize limit may experience notable to very long delays when processing those\nmessages, which may lead to a Denial of Service.\n\nAn OBJECT IDENTIFIER is composed of a series of numbers - sub-identifiers -\nmost of which have no size limit.  OBJ_obj2txt() may be used to translate\nan ASN.1 OBJECT IDENTIFIER given in DER encoding form (using the OpenSSL\ntype ASN1_OBJECT) to its canonical numeric text form, which are the\nsub-identifiers of the OBJECT IDENTIFIER in decimal form, separated by\nperiods.\n\nWhen one of the sub-identifiers in the OBJECT IDENTIFIER is very large\n(these are sizes that are seen as absurdly large, taking up tens or hundreds\nof KiBs), the translation to a decimal number in text may take a very long\ntime.  The time complexity is O(n^2) with 'n' being the size of the\nsub-identifiers in bytes (*).\n\nWith OpenSSL 3.0, support to fetch cryptographic algorithms using names /\nidentifiers in string form was introduced.  This includes using OBJECT\nIDENTIFIERs in canonical numeric text form as identifiers for fetching\nalgorithms.\n\nSuch OBJECT IDENTIFIERs may be received through the ASN.1 structure\nAlgorithmIdentifier, which is commonly used in multiple protocols to specify\nwhat cryptographic algorithm should be used to sign or verify, encrypt or\ndecrypt, or digest passed data.\n\nApplications that call OBJ_obj2txt() directly with untrusted data are\naffected, with any version of OpenSSL.  If the use is for the mere purpose\nof display, the severity is considered low.\n\nIn OpenSSL 3.0 and newer, this affects the subsystems OCSP, PKCS7/SMIME,\nCMS, CMP/CRMF or TS.  It also impacts anything that processes X.509\ncertificates, including simple things like verifying its signature.\n\nThe impact on TLS is relatively low, because all versions of OpenSSL have a\n100KiB limit on the peer's certificate chain.  Additionally, this only\nimpacts clients, or servers that have explicitly enabled client\nauthentication.\n\nIn OpenSSL 1.1.1 and 1.0.2, this only affects displaying diverse objects,\nsuch as X.509 certificates.  This is assumed to not happen in such a way\nthat it would cause a Denial of Service, so these versions are considered\nnot affected by this issue in such a way that it would be cause for concern,\nand the severity is therefore considered low."

var nvdValue1118 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1117}

var nvdValue1119 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1118}, ID: "CVE-2023-2650", LastModified: "2026-06-17T05:53:06.257", Metrics: &nvdValue1060, Published: "2023-05-30T14:15:09.683"}

var nvdValue1120 = "Allocation of Resources Without Limits or Throttling vulnerability in Apache Software Foundation Apache Struts.This issue affects Apache Struts: through 2.5.30, through 6.1.2.\n\nUpgrade to Struts 2.5.31 or 6.1.2.1 or greater."

var nvdValue1121 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1120}

var nvdValue1122 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue478}}

var nvdValue1123 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1121}, ID: "CVE-2023-34149", LastModified: "2026-06-17T06:03:00.197", Metrics: &nvdValue1122, Published: "2023-06-14T08:15:09.450"}

var nvdValue1124 = "Allocation of Resources Without Limits or Throttling vulnerability in Apache Software Foundation Apache Struts.This issue affects Apache Struts: through 2.5.30, through 6.1.2.\n\nUpgrade to Struts 2.5.31 or 6.1.2.1 or greater"

var nvdValue1125 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1124}

var nvdValue1126 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1125}, ID: "CVE-2023-34396", LastModified: "2026-06-17T06:03:33.020", Metrics: &nvdValue997, Published: "2023-06-14T08:15:09.520"}

var nvdValue1127 = "A vulnerability was found in libX11. The security flaw occurs because the functions in src/InitExt.c in libX11 do not check that the values provided for the Request, Event, or Error IDs are within the bounds of the arrays that those functions write to, using those IDs as array indexes. They trust that they were called with values provided by an Xserver adhering to the bounds specified in the X11 protocol, as all X servers provided by X.Org do. As the protocol only specifies a single byte for these values, an out-of-bounds value provided by a malicious server (or a malicious proxy-in-the-middle) can only overwrite other portions of the Display structure and not write outside the bounds of the Display structure itself, possibly causing the client to crash with this memory corruption."

var nvdValue1128 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1127}

var nvdValue1129 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1128}, ID: "CVE-2023-3138", LastModified: "2026-06-17T06:13:26.193", Metrics: &nvdValue997, Published: "2023-06-28T21:15:10.247"}

var nvdValue1130 = "The llhttp parser in the http module in Node v20.2.0 does not strictly use the CRLF sequence to delimit HTTP requests. This can lead to HTTP Request Smuggling (HRS).\r\n\r\nThe CR character (without LF) is sufficient to delimit HTTP header fields in the llhttp parser. According to RFC7230 section 3, only the CRLF sequence should delimit each header-field. This impacts all Node.js active versions: v16, v18, and, v20"

var nvdValue1131 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1130}

var nvdValue1132 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue613}}

var nvdValue1133 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1131}, ID: "CVE-2023-30589", LastModified: "2026-06-17T05:55:05.633", Metrics: &nvdValue1132, Published: "2023-07-01T00:15:10.293"}

var nvdValue1134 = "Envoy is a cloud-native high-performance edge/middle/service proxy. Envoy’s HTTP/2 codec may leak a header map and bookkeeping structures upon receiving `RST_STREAM` immediately followed by the `GOAWAY` frames from an upstream server. In nghttp2, cleanup of pending requests due to receipt of the `GOAWAY` frame skips de-allocation of the bookkeeping structure and pending compressed header. The error return [code path] is taken if connection is already marked for not sending more requests due to `GOAWAY` frame. The clean-up code is right after the return statement, causing memory leak. Denial of service through memory exhaustion. This vulnerability was patched in versions(s) 1.26.3, 1.25.8, 1.24.9, 1.23.11."

var nvdValue1135 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1134}

var nvdValue1136 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1135}, ID: "CVE-2023-35945", LastModified: "2026-06-17T06:05:31.390", Metrics: &nvdValue997, Published: "2023-07-13T21:15:08.880"}

var nvdValue1137 = "Issue summary: Checking excessively long DH keys or parameters may be very slow.\n\nImpact summary: Applications that use the functions DH_check(), DH_check_ex()\nor EVP_PKEY_param_check() to check a DH key or DH parameters may experience long\ndelays. Where the key or parameters that are being checked have been obtained\nfrom an untrusted source this may lead to a Denial of Service.\n\nThe function DH_check() performs various checks on DH parameters. One of those\nchecks confirms that the modulus ('p' parameter) is not too large. Trying to use\na very large modulus is slow and OpenSSL will not normally use a modulus which\nis over 10,000 bits in length.\n\nHowever the DH_check() function checks numerous aspects of the key or parameters\nthat have been supplied. Some of those checks use the supplied modulus value\neven if it has already been found to be too large.\n\nAn application that calls DH_check() and supplies a key or parameters obtained\nfrom an untrusted source could be vulernable to a Denial of Service attack.\n\nThe function DH_check() is itself called by a number of other OpenSSL functions.\nAn application calling any of those other functions may similarly be affected.\nThe other functions affected by this are DH_check_ex() and\nEVP_PKEY_param_check().\n\nAlso vulnerable are the OpenSSL dhparam and pkeyparam command line applications\nwhen using the '-check' option.\n\nThe OpenSSL SSL/TLS implementation is not affected by this issue.\nThe OpenSSL 3.0 and 3.1 FIPS providers are not affected by this issue."

var nvdValue1138 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1137}

var nvdValue1139 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue557}}

var nvdValue1140 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1138}, ID: "CVE-2023-3446", LastModified: "2026-06-17T06:14:05.537", Metrics: &nvdValue1139, Published: "2023-07-19T12:15:10.003"}

var nvdValue1141 = "Issue summary: Checking excessively long DH keys or parameters may be very slow.\n\nImpact summary: Applications that use the functions DH_check(), DH_check_ex()\nor EVP_PKEY_param_check() to check a DH key or DH parameters may experience long\ndelays. Where the key or parameters that are being checked have been obtained\nfrom an untrusted source this may lead to a Denial of Service.\n\nThe function DH_check() performs various checks on DH parameters. After fixing\nCVE-2023-3446 it was discovered that a large q parameter value can also trigger\nan overly long computation during some of these checks. A correct q value,\nif present, cannot be larger than the modulus p parameter, thus it is\nunnecessary to perform these checks if q is larger than p.\n\nAn application that calls DH_check() and supplies a key or parameters obtained\nfrom an untrusted source could be vulnerable to a Denial of Service attack.\n\nThe function DH_check() is itself called by a number of other OpenSSL functions.\nAn application calling any of those other functions may similarly be affected.\nThe other functions affected by this are DH_check_ex() and\nEVP_PKEY_param_check().\n\nAlso vulnerable are the OpenSSL dhparam and pkeyparam command line applications\nwhen using the \"-check\" option.\n\nThe OpenSSL SSL/TLS implementation is not affected by this issue.\n\nThe OpenSSL 3.0 and 3.1 FIPS providers are not affected by this issue."

var nvdValue1142 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1141}

var nvdValue1143 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1142}, ID: "CVE-2023-3817", LastModified: "2026-06-17T06:14:55.283", Metrics: &nvdValue1139, Published: "2023-07-31T16:15:10.497"}

var nvdValue1144 = "The use of `module.constructor.createRequire()` can bypass the policy mechanism and require modules outside of the policy.json definition for a given module.\n\nThis vulnerability affects all users using the experimental policy mechanism in all active release lines: 16.x, 18.x, and, 20.x.\n\nPlease note that at the time this CVE was issued, the policy is an experimental feature of Node.js."

var nvdValue1145 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1144}

var nvdValue1146 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1145}, ID: "CVE-2023-32006", LastModified: "2026-06-17T05:57:50.003", Metrics: &nvdValue1095, Published: "2023-08-15T16:15:11.460"}

var nvdValue1147 = "The use of `Module._load()` can bypass the policy mechanism and require modules outside of the policy.json definition for a given module.\n\nThis vulnerability affects all users using the experimental policy mechanism in all active release lines: 16.x, 18.x and, 20.x.\n\nPlease note that at the time this CVE was issued, the policy is an experimental feature of Node.js."

var nvdValue1148 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1147}

var nvdValue1149 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1148}, ID: "CVE-2023-32002", LastModified: "2026-06-17T05:57:49.263", Metrics: &nvdValue1007, Published: "2023-08-21T17:15:47.000"}

var nvdValue1150 = "A privilege escalation vulnerability exists in the experimental policy mechanism in all active release lines: 16.x, 18.x and, 20.x. The use of the deprecated API `process.binding()` can bypass the policy mechanism by requiring internal modules and eventually take advantage of `process.binding('spawn_sync')` run arbitrary code, outside of the limits defined in a `policy.json` file. Please note that at the time this CVE was issued, the policy is an experimental feature of Node.js."

var nvdValue1151 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1150}

var nvdValue1152 = schema.CVSSV31{BaseScore: 7.5, VectorString: "CVSS:3.1/AV:N/AC:H/PR:L/UI:N/S:U/C:H/I:H/A:H", Version: "3.1"}

var nvdValue1153 = schema.CVEAPIJSON20CVSSV31{CvssData: &nvdValue1152}

var nvdValue1154 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue1153}}

var nvdValue1155 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1151}, ID: "CVE-2023-32559", LastModified: "2026-06-17T05:59:07.080", Metrics: &nvdValue1154, Published: "2023-08-24T02:15:09.210"}

var nvdValue1156 = "A flaw has been identified in glibc. In an uncommon situation, the gaih_inet function may use memory that has been freed, resulting in an application crash. This issue is only exploitable when the getaddrinfo function is called and the hosts database in /etc/nsswitch.conf is configured with SUCCESS=continue or SUCCESS=merge."

var nvdValue1157 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1156}

var nvdValue1158 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1157}, ID: "CVE-2023-4813", LastModified: "2026-06-17T06:38:38.473", Metrics: &nvdValue1028, Published: "2023-09-12T22:15:08.277"}

var nvdValue1159 = "A flaw was found in glibc. When the getaddrinfo function is called with the AF_UNSPEC address family and the system is configured with no-aaaa mode via /etc/resolv.conf, a DNS response via TCP larger than 2048 bytes can potentially disclose stack contents through the function returned address data, and may cause a crash."

var nvdValue1160 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1159}

var nvdValue1161 = schema.CVSSV31{BaseScore: 6.5, VectorString: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:L/I:N/A:H", Version: "3.1"}

var nvdValue1162 = schema.CVEAPIJSON20CVSSV31{CvssData: &nvdValue1161}

var nvdValue1163 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue1162}}

var nvdValue1164 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1160}, ID: "CVE-2023-4527", LastModified: "2026-06-17T06:38:01.153", Metrics: &nvdValue1163, Published: "2023-09-18T17:15:55.067"}

var nvdValue1165 = "A flaw has been identified in glibc. In an extremely rare situation, the getaddrinfo function may access memory that has been freed, resulting in an application crash. This issue is only exploitable when a NSS module implements only the _nss_*_gethostbyname2_r and _nss_*_getcanonname_r hooks without implementing the _nss_*_gethostbyname3_r hook. The resolved name should return a large number of IPv6 and IPv4, and the call to the getaddrinfo function should have the AF_INET6 address family with AI_CANONNAME, AI_ALL and AI_V4MAPPED as flags."

var nvdValue1166 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1165}

var nvdValue1167 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1166}, ID: "CVE-2023-4806", LastModified: "2026-07-14T14:16:31.910", Metrics: &nvdValue1028, Published: "2023-09-18T17:15:55.813"}

var nvdValue1168 = "A flaw was found in the GNU C Library. A recent fix for CVE-2023-4806 introduced the potential for a memory leak, which may result in an application crash."

var nvdValue1169 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1168}

var nvdValue1170 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1169}, ID: "CVE-2023-5156", LastModified: "2026-06-17T06:47:40.620", Metrics: &nvdValue997, Published: "2023-09-25T16:15:15.613"}

var nvdValue1171 = "A buffer overflow was discovered in the GNU C Library's dynamic loader ld.so while processing the GLIBC_TUNABLES environment variable. This issue could allow a local attacker to use maliciously crafted GLIBC_TUNABLES environment variables when launching binaries with SUID permission to execute code with elevated privileges."

var nvdValue1172 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1171}

var nvdValue1173 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue449}}

var nvdValue1174 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1172}, ID: "CVE-2023-4911", LastModified: "2026-06-17T06:38:52.787", Metrics: &nvdValue1173, Published: "2023-10-03T18:15:10.463"}

var nvdValue1175 = "The HTTP/2 protocol allows a denial of service (server resource consumption) because request cancellation can reset many streams quickly, as exploited in the wild in August through October 2023."

var nvdValue1176 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1175}

var nvdValue1177 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1176}, ID: "CVE-2023-44487", LastModified: "2026-08-11T19:37:30.880", Metrics: &nvdValue997, Published: "2023-10-10T14:15:10.883"}

var nvdValue1178 = "This flaw makes curl overflow a heap based buffer in the SOCKS5 proxy\nhandshake.\n\nWhen curl is asked to pass along the host name to the SOCKS5 proxy to allow\nthat to resolve the address instead of it getting done by curl itself, the\nmaximum length that host name can be is 255 bytes.\n\nIf the host name is detected to be longer, curl switches to local name\nresolving and instead passes on the resolved address only. Due to this bug,\nthe local variable that means \"let the host resolve the name\" could get the\nwrong value during a slow SOCKS5 handshake, and contrary to the intention,\ncopy the too long host name to the target buffer instead of copying just the\nresolved address there.\n\nThe target buffer being a heap based buffer, and the host name coming from the\nURL that curl has been told to operate with."

var nvdValue1179 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1178}

var nvdValue1180 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1179}, ID: "CVE-2023-38545", LastModified: "2026-06-17T06:10:41.627", Metrics: &nvdValue1007, Published: "2023-10-18T04:15:11.077"}

var nvdValue1181 = "This flaw allows an attacker to insert cookies at will into a running program\nusing libcurl, if the specific series of conditions are met.\n\nlibcurl performs transfers. In its API, an application creates \"easy handles\"\nthat are the individual handles for single transfers.\n\nlibcurl provides a function call that duplicates en easy handle called\n[curl_easy_duphandle](https://curl.se/libcurl/c/curl_easy_duphandle.html).\n\nIf a transfer has cookies enabled when the handle is duplicated, the\ncookie-enable state is also cloned - but without cloning the actual\ncookies. If the source handle did not read any cookies from a specific file on\ndisk, the cloned version of the handle would instead store the file name as\n`none` (using the four ASCII letters, no quotes).\n\nSubsequent use of the cloned handle that does not explicitly set a source to\nload cookies from would then inadvertently load cookies from a file named\n`none` - if such a file exists and is readable in the current directory of the\nprogram using libcurl. And if using the correct file format of course."

var nvdValue1182 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1181}

var nvdValue1183 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue35}}

var nvdValue1184 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1182}, ID: "CVE-2023-38546", LastModified: "2026-06-17T06:10:41.860", Metrics: &nvdValue1183, Published: "2023-10-18T04:15:11.137"}

var nvdValue1185 = "When the Node.js policy feature checks the integrity of a resource against a trusted manifest, the application can intercept the operation and return a forged checksum to the node's policy implementation, thus effectively disabling the integrity check.\nImpacts:\nThis vulnerability affects all users using the experimental policy mechanism in all active release lines: 18.x and, 20.x.\nPlease note that at the time this CVE was issued, the policy mechanism is an experimental feature of Node.js."

var nvdValue1186 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1185}

var nvdValue1187 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1186}, ID: "CVE-2023-38552", LastModified: "2026-06-17T06:10:42.517", Metrics: &nvdValue1132, Published: "2023-10-18T04:15:11.200"}

var nvdValue1188 = "Issue summary: Generating excessively long X9.42 DH keys or checking\nexcessively long X9.42 DH keys or parameters may be very slow.\n\nImpact summary: Applications that use the functions DH_generate_key() to\ngenerate an X9.42 DH key may experience long delays.  Likewise, applications\nthat use DH_check_pub_key(), DH_check_pub_key_ex() or EVP_PKEY_public_check()\nto check an X9.42 DH key or X9.42 DH parameters may experience long delays.\nWhere the key or parameters that are being checked have been obtained from\nan untrusted source this may lead to a Denial of Service.\n\nWhile DH_check() performs all the necessary checks (as of CVE-2023-3817),\nDH_check_pub_key() doesn't make any of these checks, and is therefore\nvulnerable for excessively large P and Q parameters.\n\nLikewise, while DH_generate_key() performs a check for an excessively large\nP, it doesn't check for an excessively large Q.\n\nAn application that calls DH_generate_key() or DH_check_pub_key() and\nsupplies a key or parameters obtained from an untrusted source could be\nvulnerable to a Denial of Service attack.\n\nDH_generate_key() and DH_check_pub_key() are also called by a number of\nother OpenSSL functions.  An application calling any of those other\nfunctions may similarly be affected.  The other functions affected by this\nare DH_check_pub_key_ex(), EVP_PKEY_public_check(), and EVP_PKEY_generate().\n\nAlso vulnerable are the OpenSSL pkey command line application when using the\n\"-pubcheck\" option, as well as the OpenSSL genpkey command line application.\n\nThe OpenSSL SSL/TLS implementation is not affected by this issue.\n\nThe OpenSSL 3.0 and 3.1 FIPS providers are not affected by this issue."

var nvdValue1189 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1188}

var nvdValue1190 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1189}, ID: "CVE-2023-5678", LastModified: "2026-06-17T06:49:03.663", Metrics: &nvdValue1139, Published: "2023-11-06T16:15:42.670"}

var nvdValue1191 = "The use of __proto__ in process.mainModule.__proto__.require() can bypass the policy mechanism and require modules outside of the policy.json definition. This vulnerability affects all users using the experimental policy mechanism in all active release lines: v16, v18 and, v20.\n\nPlease note that at the time this CVE was issued, the policy is an experimental feature of Node.js"

var nvdValue1192 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1191}

var nvdValue1193 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1192}, ID: "CVE-2023-30581", LastModified: "2026-06-17T05:55:03.993", Metrics: &nvdValue1132, Published: "2023-11-23T00:15:07.980"}

var nvdValue1194 = "A vulnerability has been identified in the Node.js (.msi version) installation process, specifically affecting Windows users who install Node.js using the .msi installer. This vulnerability emerges during the repair operation, where the \"msiexec.exe\" process, running under the NT AUTHORITY\\SYSTEM context, attempts to read the %USERPROFILE% environment variable from the current user's registry.\n\nThe issue arises when the path referenced by the %USERPROFILE% environment variable does not exist. In such cases, the \"msiexec.exe\" process attempts to create the specified path in an unsafe manner, potentially leading to the creation of arbitrary folders in arbitrary locations.\n\nThe severity of this vulnerability is heightened by the fact that the %USERPROFILE% environment variable in the Windows registry can be modified by standard (or \"non-privileged\") users. Consequently, unprivileged actors, including malicious entities or trojans, can manipulate the environment variable key to deceive the privileged \"msiexec.exe\" process. This manipulation can result in the creation of folders in unintended and potentially malicious locations.\n\nIt is important to note that this vulnerability is specific to Windows users who install Node.js using the .msi installer. Users who opt for other installation methods are not affected by this particular issue."

var nvdValue1195 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1194}

var nvdValue1196 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1195}, ID: "CVE-2023-30585", LastModified: "2026-06-17T05:55:04.900", Metrics: &nvdValue1132, Published: "2023-11-28T02:15:42.077"}

var nvdValue1197 = "When an invalid public key is used to create an x509 certificate using the crypto.X509Certificate() API a non-expect termination occurs making it susceptible to DoS attacks when the attacker could force interruptions of application processing, as the process terminates when accessing public key info of provided certificates from user code. The current context of the users will be gone, and that will cause a DoS scenario. This vulnerability affects all active Node.js versions v16, v18, and, v20."

var nvdValue1198 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1197}

var nvdValue1199 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1198}, ID: "CVE-2023-30588", LastModified: "2026-06-17T05:55:05.493", Metrics: &nvdValue1139, Published: "2023-11-28T20:15:07.437"}

var nvdValue1200 = "The generateKeys() API function returned from crypto.createDiffieHellman() only generates missing (or outdated) keys, that is, it only generates a private key if none has been set yet, but the function is also needed to compute the corresponding public key after calling setPrivateKey(). However, the documentation says this API call: \"Generates private and public Diffie-Hellman key values\".\n\nThe documented behavior is very different from the actual behavior, and this difference could easily lead to security issues in applications that use these APIs as the DiffieHellman may be used as the basis for application-level security, implications are consequently broad."

var nvdValue1201 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1200}

var nvdValue1202 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1201}, ID: "CVE-2023-30590", LastModified: "2026-06-17T05:55:05.810", Metrics: &nvdValue1132, Published: "2023-11-28T20:15:07.480"}

var nvdValue1203 = "When a Multipart request is performed but some of the fields exceed the maxStringLength\u00a0 limit, the upload files will remain in struts.multipart.saveDir\u00a0 even if the request has been denied.\nUsers are recommended to upgrade to versions Struts 2.5.32 or 6.1.2.2 or Struts 6.3.0.1 or greater, which fixe this issue."

var nvdValue1204 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1203}

var nvdValue1205 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1204}, ID: "CVE-2023-41835", LastModified: "2026-06-17T06:22:54.200", Metrics: &nvdValue997, Published: "2023-12-05T09:15:07.093"}

var nvdValue1206 = "This flaw allows a malicious HTTP server to set \"super cookies\" in curl that\nare then passed back to more origins than what is otherwise allowed or\npossible. This allows a site to set cookies that then would get sent to\ndifferent and unrelated sites and domains.\n\nIt could do this by exploiting a mixed case flaw in curl's function that\nverifies a given cookie domain against the Public Suffix List (PSL). For\nexample a cookie could be set with `domain=co.UK` when the URL used a lower\ncase hostname `curl.co.uk`, even though `co.uk` is listed as a PSL domain."

var nvdValue1207 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1206}

var nvdValue1208 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1207}, ID: "CVE-2023-46218", LastModified: "2026-06-17T06:30:22.953", Metrics: &nvdValue967, Published: "2023-12-07T01:15:07.160"}

var nvdValue1209 = "An attacker can manipulate file upload params to enable paths traversal and under some circumstances this can lead to uploading a malicious file which can be used to perform Remote Code Execution.\nUsers are recommended to upgrade to versions Struts 2.5.33 or Struts 6.3.0.2 or greater to\u00a0fix this issue."

var nvdValue1210 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1209}

var nvdValue1211 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1210}, ID: "CVE-2023-50164", LastModified: "2026-06-17T06:39:13.613", Metrics: &nvdValue1007, Published: "2023-12-07T09:15:07.060"}

var nvdValue1212 = "The SSH transport protocol with certain OpenSSH extensions, found in OpenSSH before 9.6 and other products, allows remote attackers to bypass integrity checks such that some packets are omitted (from the extension negotiation message), and a client and server may consequently end up with a connection for which some security features have been downgraded or disabled, aka a Terrapin attack. This occurs because the SSH Binary Packet Protocol (BPP), implemented by these extensions, mishandles the handshake phase and mishandles use of sequence numbers. For example, there is an effective attack against SSH's use of ChaCha20-Poly1305 (and CBC with Encrypt-then-MAC). The bypass occurs in chacha20-poly1305@openssh.com and (if CBC is used) the -etm@openssh.com MAC algorithms. This also affects Maverick Synergy Java SSH API before 3.1.0-SNAPSHOT, Dropbear through 2022.83, Ssh before 5.1.1 in Erlang/OTP, PuTTY before 0.80, AsyncSSH before 2.14.2, golang.org/x/crypto before 0.17.0, libssh before 0.10.6, libssh2 through 1.11.0, Thorn Tech SFTP Gateway before 3.4.6, Tera Term before 5.1, Paramiko before 3.4.0, jsch before 0.2.15, SFTPGo before 2.5.6, Netgate pfSense Plus through 23.09.1, Netgate pfSense CE through 2.7.2, HPN-SSH through 18.2.0, ProFTPD before 1.3.8b (and before 1.3.9rc2), ORYX CycloneSSH before 2.3.4, NetSarang XShell 7 before Build 0144, CrushFTP before 10.6.0, ConnectBot SSH library before 2.2.22, Apache MINA sshd through 2.11.0, sshj through 0.37.0, TinySSH through 20230101, trilead-ssh2 6401, LANCOM LCOS and LANconfig, FileZilla before 3.66.4, Nova before 11.8, PKIX-SSH before 14.4, SecureCRT before 9.4.3, Transmit5 before 5.10.4, Win32-OpenSSH before 9.5.0.0p1-Beta, WinSCP before 6.2.2, Bitvise SSH Server before 9.32, Bitvise SSH Client before 9.33, KiTTY through 0.76.1.13, the net-ssh gem 7.2.0 for Ruby, the mscdex ssh2 module before 1.15.0 for Node.js, the thrussh library before 0.35.1 for Rust, and the Russh crate before 0.40.2 for Rust."

var nvdValue1213 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1212}

var nvdValue1214 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1213}, ID: "CVE-2023-48795", LastModified: "2026-06-17T06:34:59.200", Metrics: &nvdValue1111, Published: "2023-12-18T16:15:10.897"}

var nvdValue1215 = "A vulnerability was found in systemd-resolved. This issue may allow systemd-resolved to accept records of DNSSEC-signed domains even when they have no signature, allowing man-in-the-middles (or the upstream DNS resolver) to manipulate records."

var nvdValue1216 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1215}

var nvdValue1217 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1216}, ID: "CVE-2023-7008", LastModified: "2026-06-17T06:51:51.530", Metrics: &nvdValue1111, Published: "2023-12-23T13:15:07.573"}

var nvdValue1218 = "A vulnerability was found in SQLite SQLite3 up to 3.43.0 and classified as critical. This issue affects the function sessionReadRecord of the file ext/session/sqlite3session.c of the component make alltest Handler. The manipulation leads to heap-based buffer overflow. It is recommended to apply a patch to fix this issue. The associated identifier of this vulnerability is VDB-248999."

var nvdValue1219 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1218}

var nvdValue1220 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue529}}

var nvdValue1221 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1219}, ID: "CVE-2023-7104", LastModified: "2026-06-17T06:52:04.587", Metrics: &nvdValue1220, Published: "2023-12-29T10:15:13.890"}

var nvdValue1222 = "A heap-based buffer overflow was found in the __vsyslog_internal function of the glibc library. This function is called by the syslog and vsyslog functions. This issue occurs when the openlog function was not called, or called with the ident argument set to NULL, and the program name (the basename of argv[0]) is bigger than 1024 bytes, resulting in an application crash or local privilege escalation. This issue affects glibc 2.36 and newer."

var nvdValue1223 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1222}

var nvdValue1224 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1223}, ID: "CVE-2023-6246", LastModified: "2026-06-17T06:50:23.203", Metrics: &nvdValue1173, Published: "2024-01-31T14:15:48.420"}

var nvdValue1225 = "An off-by-one heap-based buffer overflow was found in the __vsyslog_internal function of the glibc library. This function is called by the syslog and vsyslog functions. This issue occurs when these functions are called with a message bigger than INT_MAX bytes, leading to an incorrect calculation of the buffer size to store the message, resulting in an application crash. This issue affects glibc 2.37 and newer."

var nvdValue1226 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1225}

var nvdValue1227 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1226}, ID: "CVE-2023-6779", LastModified: "2026-06-17T06:51:24.530", Metrics: &nvdValue997, Published: "2024-01-31T14:15:48.700"}

var nvdValue1228 = "libexpat through 2.5.0 allows a denial of service (resource consumption) because many full reparsings are required in the case of a large token for which multiple buffer fills are needed."

var nvdValue1229 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1228}

var nvdValue1230 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1229}, ID: "CVE-2023-52425", LastModified: "2026-06-17T06:42:39.490", Metrics: &nvdValue997, Published: "2024-02-04T20:15:46.063"}

var nvdValue1231 = "Maliciously crafted export names in an imported WebAssembly module can inject JavaScript code. The injected code may be able to access data and functions that the WebAssembly module itself does not have access to, similar to as if the WebAssembly module was a JavaScript module.\n\nThis vulnerability affects users of any active release line of Node.js. The vulnerable feature is only available if Node.js is started with the `--experimental-wasm-modules` command line option."

var nvdValue1232 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1231}

var nvdValue1233 = schema.CVEAPIJSON20CVEItemMetrics{}

var nvdValue1234 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1232}, ID: "CVE-2023-39333", LastModified: "2026-06-17T06:12:03.327", Metrics: &nvdValue1233, Published: "2024-09-07T16:15:02.287"}

var nvdValue1235 = "Issue summary: Processing a maliciously formatted PKCS12 file may lead OpenSSL\nto crash leading to a potential Denial of Service attack\n\nImpact summary: Applications loading files in the PKCS12 format from untrusted\nsources might terminate abruptly.\n\nA file in PKCS12 format can contain certificates and keys and may come from an\nuntrusted source. The PKCS12 specification allows certain fields to be NULL, but\nOpenSSL does not correctly check for this case. This can lead to a NULL pointer\ndereference that results in OpenSSL crashing. If an application processes PKCS12\nfiles from an untrusted source using the OpenSSL APIs then that application will\nbe vulnerable to this issue.\n\nOpenSSL APIs that are vulnerable to this are: PKCS12_parse(),\nPKCS12_unpack_p7data(), PKCS12_unpack_p7encdata(), PKCS12_unpack_authsafes()\nand PKCS12_newpass().\n\nWe have also fixed a similar issue in SMIME_write_PKCS7(). However since this\nfunction is related to writing data we do not consider it security significant.\n\nThe FIPS modules in 3.2, 3.1 and 3.0 are not affected by this issue."

var nvdValue1236 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1235}

var nvdValue1237 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue290}}

var nvdValue1238 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1236}, ID: "CVE-2024-0727", LastModified: "2026-06-17T06:54:07.030", Metrics: &nvdValue1237, Published: "2024-01-26T09:15:07.637"}

var nvdValue1239 = "libexpat through 2.6.1 allows an XML Entity Expansion attack when there is isolated use of external parsers (created via XML_ExternalEntityParserCreate)."

var nvdValue1240 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1239}

var nvdValue1241 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1240}, ID: "CVE-2024-28757", LastModified: "2026-06-17T07:21:45.160", Metrics: &nvdValue997, Published: "2024-03-10T05:15:06.570"}

var nvdValue1242 = "When an application tells libcurl it wants to allow HTTP/2 server push, and the amount of received headers for the push surpasses the maximum allowed limit (1000), libcurl aborts the server push. When aborting, libcurl inadvertently does not free all the previously allocated headers and instead leaks the memory.  Further, this error condition fails silently and is therefore not easily detected by an application."

var nvdValue1243 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1242}

var nvdValue1244 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1243}, ID: "CVE-2024-2398", LastModified: "2026-06-17T07:24:27.170", Metrics: &nvdValue1233, Published: "2024-03-27T08:15:41.283"}

var nvdValue1245 = "Issue summary: Some non-default TLS server configurations can cause unbounded\nmemory growth when processing TLSv1.3 sessions\n\nImpact summary: An attacker may exploit certain server configurations to trigger\nunbounded memory growth that would lead to a Denial of Service\n\nThis problem can occur in TLSv1.3 if the non-default SSL_OP_NO_TICKET option is\nbeing used (but not if early_data support is also configured and the default\nanti-replay protection is in use). In this case, under certain conditions, the\nsession cache can get into an incorrect state and it will fail to flush properly\nas it fills. The session cache will continue to grow in an unbounded manner. A\nmalicious client could deliberately create the scenario for this failure to\nforce a Denial of Service. It may also happen by accident in normal operation.\n\nThis issue only affects TLS servers supporting TLSv1.3. It does not affect TLS\nclients.\n\nThe FIPS modules in 3.2, 3.1 and 3.0 are not affected by this issue. OpenSSL\n1.0.2 is also not affected by this issue."

var nvdValue1246 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1245}

var nvdValue1247 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1246}, ID: "CVE-2024-2511", LastModified: "2026-06-17T07:24:40.830", Metrics: &nvdValue1233, Published: "2024-04-08T14:15:07.660"}

var nvdValue1248 = "A command inject vulnerability allows an attacker to perform command injection on Windows applications that indirectly depend on the CreateProcess function when the specific conditions are satisfied."

var nvdValue1249 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1248}

var nvdValue1250 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1249}, ID: "CVE-2024-3566", LastModified: "2026-06-17T07:44:31.710", Metrics: &nvdValue1007, Published: "2024-04-10T16:15:16.083"}

var nvdValue1251 = "The iconv() function in the GNU C Library versions 2.39 and older may overflow the output buffer passed to it by up to 4 bytes when converting strings to the ISO-2022-CN-EXT character set, which may be used to crash an application or overwrite a neighbouring variable."

var nvdValue1252 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1251}

var nvdValue1253 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1252}, ID: "CVE-2024-2961", LastModified: "2026-06-17T07:25:54.767", Metrics: &nvdValue1233, Published: "2024-04-17T18:15:15.833"}

var nvdValue1254 = "nscd: Stack-based buffer overflow in netgroup cache\n\nIf the Name Service Cache Daemon's (nscd) fixed size cache is exhausted\nby client requests then a subsequent client request for netgroup data\nmay result in a stack-based buffer overflow.  This flaw was introduced\nin glibc 2.15 when the cache was added to nscd.\n\nThis vulnerability is only present in the nscd binary."

var nvdValue1255 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1254}

var nvdValue1256 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1255}, ID: "CVE-2024-33599", LastModified: "2026-06-17T07:32:03.820", Metrics: &nvdValue1233, Published: "2024-05-06T20:15:11.437"}

var nvdValue1257 = "nscd: netgroup cache may terminate daemon on memory allocation failure\n\nThe Name Service Cache Daemon's (nscd) netgroup cache uses xmalloc or\nxrealloc and these functions may terminate the process due to a memory\nallocation failure resulting in a denial of service to the clients.  The\nflaw was introduced in glibc 2.15 when the cache was added to nscd.\n\nThis vulnerability is only present in the nscd binary."

var nvdValue1258 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1257}

var nvdValue1259 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1258}, ID: "CVE-2024-33601", LastModified: "2026-06-17T07:32:04.173", Metrics: &nvdValue1233, Published: "2024-05-06T20:15:11.603"}

var nvdValue1260 = "nscd: netgroup cache assumes NSS callback uses in-buffer strings\n\nThe Name Service Cache Daemon's (nscd) netgroup cache can corrupt memory\nwhen the NSS callback does not store all strings in the provided buffer.\nThe flaw was introduced in glibc 2.15 when the cache was added to nscd.\n\nThis vulnerability is only present in the nscd binary."

var nvdValue1261 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1260}

var nvdValue1262 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1261}, ID: "CVE-2024-33602", LastModified: "2026-06-17T07:32:04.343", Metrics: &nvdValue1233, Published: "2024-05-06T20:15:11.680"}

var nvdValue1263 = "Issue summary: Calling the OpenSSL API function SSL_select_next_proto with an\nempty supported client protocols buffer may cause a crash or memory contents to\nbe sent to the peer.\n\nImpact summary: A buffer overread can have a range of potential consequences\nsuch as unexpected application beahviour or a crash. In particular this issue\ncould result in up to 255 bytes of arbitrary private data from memory being sent\nto the peer leading to a loss of confidentiality. However, only applications\nthat directly call the SSL_select_next_proto function with a 0 length list of\nsupported client protocols are affected by this issue. This would normally never\nbe a valid scenario and is typically not under attacker control but may occur by\naccident in the case of a configuration or programming error in the calling\napplication.\n\nThe OpenSSL API function SSL_select_next_proto is typically used by TLS\napplications that support ALPN (Application Layer Protocol Negotiation) or NPN\n(Next Protocol Negotiation). NPN is older, was never standardised and\nis deprecated in favour of ALPN. We believe that ALPN is significantly more\nwidely deployed than NPN. The SSL_select_next_proto function accepts a list of\nprotocols from the server and a list of protocols from the client and returns\nthe first protocol that appears in the server list that also appears in the\nclient list. In the case of no overlap between the two lists it returns the\nfirst item in the client list. In either case it will signal whether an overlap\nbetween the two lists was found. In the case where SSL_select_next_proto is\ncalled with a zero length client list it fails to notice this condition and\nreturns the memory immediately following the client list pointer (and reports\nthat there was no overlap in the lists).\n\nThis function is typically called from a server side application callback for\nALPN or a client side application callback for NPN. In the case of ALPN the list\nof protocols supplied by the client is guaranteed by libssl to never be zero in\nlength. The list of server protocols comes from the application and should never\nnormally be expected to be of zero length. In this case if the\nSSL_select_next_proto function has been called as expected (with the list\nsupplied by the client passed in the client/client_len parameters), then the\napplication will not be vulnerable to this issue. If the application has\naccidentally been configured with a zero length server list, and has\naccidentally passed that zero length server list in the client/client_len\nparameters, and has additionally failed to correctly handle a \"no overlap\"\nresponse (which would normally result in a handshake failure in ALPN) then it\nwill be vulnerable to this problem.\n\nIn the case of NPN, the protocol permits the client to opportunistically select\na protocol when there is no overlap. OpenSSL returns the first client protocol\nin the no overlap case in support of this. The list of client protocols comes\nfrom the application and should never normally be expected to be of zero length.\nHowever if the SSL_select_next_proto function is accidentally called with a\nclient_len of 0 then an invalid memory pointer will be returned instead. If the\napplication uses this output as the opportunistic protocol then the loss of\nconfidentiality will occur.\n\nThis issue has been assessed as Low severity because applications are most\nlikely to be vulnerable if they are using NPN instead of ALPN - but NPN is not\nwidely used. It also requires an application configuration or programming error.\nFinally, this issue would not typically be under attacker control making active\nexploitation unlikely.\n\nThe FIPS modules in 3.3, 3.2, 3.1 and 3.0 are not affected by this issue.\n\nDue to the low severity of this issue we are not issuing new releases of\nOpenSSL at this time. The fix will be included in the next releases when they\nbecome available."

var nvdValue1264 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1263}

var nvdValue1265 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1264}, ID: "CVE-2024-5535", LastModified: "2026-06-17T08:16:08.843", Metrics: &nvdValue1233, Published: "2024-06-27T11:15:24.447"}

var nvdValue1266 = "libcurl's ASN1 parser code has the `GTime2str()` function, used for parsing an\nASN.1 Generalized Time field. If given an syntactically incorrect field, the\nparser might end up using -1 for the length of the *time fraction*, leading to\na `strlen()` getting performed on a pointer to a heap buffer area that is not\n(purposely) null terminated.\n\nThis flaw most likely leads to a crash, but can also lead to heap contents\ngetting returned to the application when\n[CURLINFO_CERTINFO](https://curl.se/libcurl/c/CURLINFO_CERTINFO.html) is used."

var nvdValue1267 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1266}

var nvdValue1268 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1267}, ID: "CVE-2024-7264", LastModified: "2026-06-17T08:19:43.880", Metrics: &nvdValue1060, Published: "2024-07-31T08:15:02.657"}

var nvdValue1269 = schema.CVEAPIJSON20LangString{Lang: "en", Value: "An issue was discovered in libexpat before 2.6.3. xmlparse.c does not reject a negative length for XML_ParseBuffer."}

var nvdValue1270 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1269}, ID: "CVE-2024-45490", LastModified: "2026-06-17T07:54:18.503", Metrics: &nvdValue997, Published: "2024-08-30T03:15:03.757"}

var nvdValue1271 = "An issue was discovered in libexpat before 2.6.3. dtdCopy in xmlparse.c can have an integer overflow for nDefaultAtts on 32-bit platforms (where UINT_MAX equals SIZE_MAX)."

var nvdValue1272 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1271}

var nvdValue1273 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1272}, ID: "CVE-2024-45491", LastModified: "2026-06-17T07:54:18.673", Metrics: &nvdValue1007, Published: "2024-08-30T03:15:03.850"}

var nvdValue1274 = "An issue was discovered in libexpat before 2.6.3. nextScaffoldPart in xmlparse.c can have an integer overflow for m_groupSize on 32-bit platforms (where UINT_MAX equals SIZE_MAX)."

var nvdValue1275 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1274}

var nvdValue1276 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1275}, ID: "CVE-2024-45492", LastModified: "2026-06-17T07:54:18.840", Metrics: &nvdValue1007, Published: "2024-08-30T03:15:03.930"}

var nvdValue1277 = "When curl is told to use the Certificate Status Request TLS extension, often referred to as OCSP stapling, to verify that the server certificate is valid, it might fail to detect some OCSP problems and instead wrongly consider the response as fine.  If the returned status reports another error than 'revoked' (like for example 'unauthorized') it is not treated as a bad certficate."

var nvdValue1278 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1277}

var nvdValue1279 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1278}, ID: "CVE-2024-8096", LastModified: "2026-06-17T08:21:51.890", Metrics: &nvdValue1233, Published: "2024-09-11T10:15:02.883"}

var nvdValue1280 = "Issue summary: Use of the low-level GF(2^m) elliptic curve APIs with untrusted\nexplicit values for the field polynomial can lead to out-of-bounds memory reads\nor writes.\n\nImpact summary: Out of bound memory writes can lead to an application crash or\neven a possibility of a remote code execution, however, in all the protocols\ninvolving Elliptic Curve Cryptography that we're aware of, either only \"named\ncurves\" are supported, or, if explicit curve parameters are supported, they\nspecify an X9.62 encoding of binary (GF(2^m)) curves that can't represent\nproblematic input values. Thus the likelihood of existence of a vulnerable\napplication is low.\n\nIn particular, the X9.62 encoding is used for ECC keys in X.509 certificates,\nso problematic inputs cannot occur in the context of processing X.509\ncertificates.  Any problematic use-cases would have to be using an \"exotic\"\ncurve encoding.\n\nThe affected APIs include: EC_GROUP_new_curve_GF2m(), EC_GROUP_new_from_params(),\nand various supporting BN_GF2m_*() functions.\n\nApplications working with \"exotic\" explicit binary (GF(2^m)) curve parameters,\nthat make it possible to represent invalid field polynomials with a zero\nconstant term, via the above or similar APIs, may terminate abruptly as a\nresult of reading or writing outside of array bounds.  Remote code execution\ncannot easily be ruled out.\n\nThe FIPS modules in 3.3, 3.2, 3.1 and 3.0 are not affected by this issue."

var nvdValue1281 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1280}

var nvdValue1282 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1281}, ID: "CVE-2024-9143", LastModified: "2026-06-17T08:24:02.610", Metrics: &nvdValue1233, Published: "2024-10-16T17:15:18.130"}

var nvdValue1283 = "An issue was discovered in libexpat before 2.6.4. There is a crash within the XML_ResumeParser function because XML_StopParser can stop/suspend an unstarted parser."

var nvdValue1284 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1283}

var nvdValue1285 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1284}, ID: "CVE-2024-50602", LastModified: "2026-06-17T08:04:47.443", Metrics: &nvdValue1233, Published: "2024-10-27T05:15:04.090"}

var nvdValue1286 = "Issue summary: Calling the OpenSSL API function SSL_free_buffers may cause\nmemory to be accessed that was previously freed in some situations\n\nImpact summary: A use after free can have a range of potential consequences such\nas the corruption of valid data, crashes or execution of arbitrary code.\nHowever, only applications that directly call the SSL_free_buffers function are\naffected by this issue. Applications that do not call this function are not\nvulnerable. Our investigations indicate that this function is rarely used by\napplications.\n\nThe SSL_free_buffers function is used to free the internal OpenSSL buffer used\nwhen processing an incoming record from the network. The call is only expected\nto succeed if the buffer is not currently in use. However, two scenarios have\nbeen identified where the buffer is freed even when still in use.\n\nThe first scenario occurs where a record header has been received from the\nnetwork and processed by OpenSSL, but the full record body has not yet arrived.\nIn this case calling SSL_free_buffers will succeed even though a record has only\nbeen partially processed and the buffer is still in use.\n\nThe second scenario occurs where a full record containing application data has\nbeen received and processed by OpenSSL but the application has only read part of\nthis data. Again a call to SSL_free_buffers will succeed even though the buffer\nis still in use.\n\nWhile these scenarios could occur accidentally during normal operation a\nmalicious attacker could attempt to engineer a stituation where this occurs.\nWe are not aware of this issue being actively exploited.\n\nThe FIPS modules in 3.3, 3.2, 3.1 and 3.0 are not affected by this issue."

var nvdValue1287 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1286}

var nvdValue1288 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1287}, ID: "CVE-2024-4741", LastModified: "2026-06-17T08:02:32.007", Metrics: &nvdValue1233, Published: "2024-11-13T11:15:04.480"}

var nvdValue1289 = "When asked to both use a `.netrc` file for credentials and to follow HTTP\nredirects, curl could leak the password used for the first host to the\nfollowed-to host under certain circumstances.\n\nThis flaw only manifests itself if the netrc file has an entry that matches\nthe redirect target hostname but the entry either omits just the password or\nomits both login and password."

var nvdValue1290 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1289}

var nvdValue1291 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1290}, ID: "CVE-2024-11053", LastModified: "2026-06-17T06:56:57.873", Metrics: &nvdValue1233, Published: "2024-12-11T08:15:05.307"}

var nvdValue1292 = "File upload logic in Apache Struts is flawed.\u00a0An attacker can manipulate file upload params to enable paths traversal and under some circumstances this can lead to uploading a malicious file which can be used to perform Remote Code Execution.\n\nThis issue affects Apache Struts: from 2.0.0 before 6.4.0.\n\nUsers are recommended to upgrade to version 6.4.0 at least and migrate to the new  file upload mechanism https://struts.apache.org/core-developers/file-upload . If you are not using an old file upload logic based on\u00a0FileuploadInterceptor\u00a0your application is safe.\n\nYou can find more details in\u00a0 https://cwiki.apache.org/confluence/display/WW/S2-067"

var nvdValue1293 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1292}

var nvdValue1294 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1293}, ID: "CVE-2024-53677", LastModified: "2026-06-17T08:09:05.010", Metrics: &nvdValue1007, Published: "2024-12-11T16:15:14.593"}

var nvdValue1295 = "Applications and libraries which misuse connection.serverAuthenticate (via callback field ServerConfig.PublicKeyCallback) may be susceptible to an authorization bypass. The documentation for ServerConfig.PublicKeyCallback says that \"A call to this function does not guarantee that the key offered is in fact used to authenticate.\" Specifically, the SSH protocol allows clients to inquire about whether a public key is acceptable before proving control of the corresponding private key. PublicKeyCallback may be called with multiple keys, and the order in which the keys were provided cannot be used to infer which key the client successfully authenticated with, if any. Some applications, which store the key(s) passed to PublicKeyCallback (or derived information) and make security relevant determinations based on it once the connection is established, may make incorrect assumptions. For example, an attacker may send public keys A and B, and then authenticate with A. PublicKeyCallback would be called only twice, first with A and then with B. A vulnerable application may then make authorization decisions based on key B for which the attacker does not actually control the private key. Since this API is widely misused, as a partial mitigation golang.org/x/cry...@v0.31.0 enforces the property that, when successfully authenticating via public key, the last key passed to ServerConfig.PublicKeyCallback will be the key used to authenticate the connection. PublicKeyCallback will now be called multiple times with the same key, if necessary. Note that the client may still not control the last key passed to PublicKeyCallback if the connection is then authenticated with a different method, such as PasswordCallback, KeyboardInteractiveCallback, or NoClientAuth. Users should be using the Extensions field of the Permissions return value from the various authentication callbacks to record data associated with the authentication attempt instead of referencing external state. Once the connection is established the state corresponding to the successful authentication attempt can be retrieved via the ServerConn.Permissions field. Note that some third-party libraries misuse the Permissions type by sharing it across authentication attempts; users of third-party libraries should refer to the relevant projects for guidance."

var nvdValue1296 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1295}

var nvdValue1297 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1296}, ID: "CVE-2024-45337", LastModified: "2026-06-17T07:54:03.360", Metrics: &nvdValue1233, Published: "2024-12-12T02:02:07.970"}

var nvdValue1298 = "Issue summary: A timing side-channel which could potentially allow recovering\nthe private key exists in the ECDSA signature computation.\n\nImpact summary: A timing side-channel in ECDSA signature computations\ncould allow recovering the private key by an attacker. However, measuring\nthe timing would require either local access to the signing application or\na very fast network connection with low latency.\n\nThere is a timing signal of around 300 nanoseconds when the top word of\nthe inverted ECDSA nonce value is zero. This can happen with significant\nprobability only for some of the supported elliptic curves. In particular\nthe NIST P-521 curve is affected. To be able to measure this leak, the attacker\nprocess must either be located in the same physical computer or must\nhave a very fast network connection with low latency. For that reason\nthe severity of this vulnerability is Low.\n\nThe FIPS modules in 3.4, 3.3, 3.2, 3.1 and 3.0 are affected by this issue."

var nvdValue1299 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1298}

var nvdValue1300 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1299}, ID: "CVE-2024-13176", LastModified: "2026-06-17T07:01:23.167", Metrics: &nvdValue1233, Published: "2025-01-20T14:15:26.247"}

var nvdValue1301 = "A stack overflow vulnerability exists in the libexpat library due to the way it handles recursive entity expansion in XML documents. When parsing an XML document with deeply nested entity references, libexpat can be forced to recurse indefinitely, exhausting the stack space and causing a crash. This issue could lead to denial of service (DoS) or, in some cases, exploitable memory corruption, depending on the environment and library usage."

var nvdValue1302 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1301}

var nvdValue1303 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1302}, ID: "CVE-2024-8176", LastModified: "2026-09-21T06:16:58.990", Metrics: &nvdValue1233, Published: "2025-03-14T09:15:14.157"}

var nvdValue1304 = "With the aid of the diagnostics_channel utility, an event can be hooked into whenever a worker thread is created. This is not limited only to workers but also exposes internal workers, where an instance of them can be fetched, and its constructor can be grabbed and reinstated for malicious usage. \r\n\r\nThis vulnerability affects Permission Model users (--permission) on Node.js v20, v22, and v23."

var nvdValue1305 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1304}

var nvdValue1306 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1305}, ID: "CVE-2025-23083", LastModified: "2026-06-17T08:51:52.600", Metrics: &nvdValue1233, Published: "2025-01-22T02:15:33.930"}

var nvdValue1307 = "A vulnerability has been identified in Node.js, specifically affecting the handling of drive names in the Windows environment. Certain Node.js functions do not treat drive names as special on Windows. As a result, although Node.js assumes a relative path, it actually refers to the root directory.\r\n\r\nOn Windows, a path that does not start with the file separator is treated as relative to the current directory. \r\n\r\nThis vulnerability affects Windows users of `path.join` API."

var nvdValue1308 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1307}

var nvdValue1309 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1308}, ID: "CVE-2025-23084", LastModified: "2026-06-17T08:51:53.080", Metrics: &nvdValue1107, Published: "2025-01-28T05:15:11.267"}

var nvdValue1310 = "A memory leak could occur when a remote peer abruptly closes the socket without sending a GOAWAY notification. Additionally, if an invalid header was detected by nghttp2, causing the connection to be terminated by the peer, the same leak was triggered. This flaw could lead to increased memory consumption and potential denial of service under certain conditions.\r\n\r\nThis vulnerability affects HTTP/2 Server users on Node.js v18.x, v20.x, v22.x and v23.x."

var nvdValue1311 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1310}

var nvdValue1312 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1311}, ID: "CVE-2025-23085", LastModified: "2026-06-17T08:51:53.580", Metrics: &nvdValue1233, Published: "2025-02-07T07:15:15.810"}

var nvdValue1313 = "SSH servers which implement file transfer protocols are vulnerable to a denial of service attack from clients which complete the key exchange slowly, or not at all, causing pending content to be read into memory, but never transmitted."

var nvdValue1314 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1313}

var nvdValue1315 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1314}, ID: "CVE-2025-22869", LastModified: "2026-06-17T08:50:40.113", Metrics: &nvdValue1233, Published: "2025-02-26T08:14:24.997"}

var nvdValue1316 = "Path Equivalence: 'file.Name' (Internal Dot) leading to\u00a0Remote Code Execution and/or Information disclosure\u00a0and/or malicious content added to uploaded files via write enabled\u00a0Default Servlet\u00a0in Apache Tomcat.\n\nThis issue affects Apache Tomcat: from 11.0.0-M1 through 11.0.2, from 10.1.0-M1 through 10.1.34, from 9.0.0.M1 through 9.0.98.\nThe following versions were EOL at the time the CVE was created but are \nknown to be affected: 8.5.0 though 8.5.100. Other, older, EOL versions \nmay also be affected.\n\n\nIf all of the following were true, a malicious user was able to view       security sensitive files and/or inject content into those files:\n-\u00a0writes enabled for the default servlet (disabled by default)\n- support for partial PUT (enabled by default)\n- a target URL for security sensitive uploads that was a sub-directory of\u00a0a target URL for public uploads\n-\u00a0attacker knowledge of the names of security sensitive files being\u00a0uploaded\n-\u00a0the security sensitive files also being uploaded via partial PUT\n\nIf all of the following were true, a malicious user was able to       perform remote code execution:\n- writes enabled for the default servlet (disabled by default)\n-\u00a0support for partial PUT (enabled by default)\n-\u00a0application was using Tomcat's file based session persistence with the\u00a0default storage location\n-\u00a0application included a library that may be leveraged in a\u00a0deserialization attack\n\nUsers are recommended to upgrade to version 11.0.3, 10.1.35 or 9.0.99, which fixes the issue."

var nvdValue1317 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1316}

var nvdValue1318 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1317}, ID: "CVE-2025-24813", LastModified: "2026-06-17T08:59:39.593", Metrics: &nvdValue1007, Published: "2025-03-10T17:15:35.067"}

var nvdValue1319 = "Untrusted LD_LIBRARY_PATH environment variable vulnerability in the GNU C Library version 2.27 to 2.38 allows attacker controlled loading of dynamically shared library in statically compiled setuid binaries that call dlopen (including internal dlopen calls after setlocale or calls to NSS functions such as getaddrinfo)."

var nvdValue1320 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1319}

var nvdValue1321 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1320}, ID: "CVE-2025-4802", LastModified: "2026-06-17T09:34:02.647", Metrics: &nvdValue1233, Published: "2025-05-16T20:15:22.280"}

var nvdValue1322 = "The html.parser.HTMLParser class had worse-case quadratic complexity when processing certain crafted malformed inputs potentially leading to amplified denial-of-service."

var nvdValue1323 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1322}

var nvdValue1324 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1323}, ID: "CVE-2025-6069", LastModified: "2026-07-31T14:16:45.130", Metrics: &nvdValue1233, Published: "2025-06-17T14:15:33.677"}

var nvdValue1325 = "The regcomp function in the GNU C library version from 2.4 to 2.41 is \nsubject to a double free if some previous allocation fails. It can be \naccomplished either by a malloc failure or by using an interposed malloc\n that injects random malloc failures. The double free can allow buffer \nmanipulation depending of how the regex is constructed. This issue \naffects all architectures and ABIs supported by the GNU C library."

var nvdValue1326 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1325}

var nvdValue1327 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1326}, ID: "CVE-2025-8058", LastModified: "2026-06-17T10:06:13.930", Metrics: &nvdValue1233, Published: "2025-07-23T20:15:27.747"}

var nvdValue1328 = "libexpat in Expat before 2.7.2 allows attackers to trigger large dynamic memory allocations via a small document that is submitted for parsing."

var nvdValue1329 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1328}

var nvdValue1330 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1329}, ID: "CVE-2025-59375", LastModified: "2026-06-17T09:46:02.697", Metrics: &nvdValue1233, Published: "2025-09-15T03:15:40.920"}

var nvdValue1331 = "Issue summary: An application trying to decrypt CMS messages encrypted using\npassword based encryption can trigger an out-of-bounds read and write.\n\nImpact summary: This out-of-bounds read may trigger a crash which leads to\nDenial of Service for an application. The out-of-bounds write can cause\na memory corruption which can have various consequences including\na Denial of Service or Execution of attacker-supplied code.\n\nAlthough the consequences of a successful exploit of this vulnerability\ncould be severe, the probability that the attacker would be able to\nperform it is low. Besides, password based (PWRI) encryption support in CMS\nmessages is very rarely used. For that reason the issue was assessed as\nModerate severity according to our Security Policy.\n\nThe FIPS modules in 3.5, 3.4, 3.3, 3.2, 3.1 and 3.0 are not affected by this\nissue, as the CMS implementation is outside the OpenSSL FIPS module\nboundary."

var nvdValue1332 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1331}

var nvdValue1333 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1332}, ID: "CVE-2025-9230", LastModified: "2026-07-14T13:18:07.327", Metrics: &nvdValue1233, Published: "2025-09-30T14:15:41.050"}

var nvdValue1334 = "Issue summary: An application using the OpenSSL HTTP client API functions may\ntrigger an out-of-bounds read if the 'no_proxy' environment variable is set and\nthe host portion of the authority component of the HTTP URL is an IPv6 address.\n\nImpact summary: An out-of-bounds read can trigger a crash which leads to\nDenial of Service for an application.\n\nThe OpenSSL HTTP client API functions can be used directly by applications\nbut they are also used by the OCSP client functions and CMP (Certificate\nManagement Protocol) client implementation in OpenSSL. However the URLs used\nby these implementations are unlikely to be controlled by an attacker.\n\nIn this vulnerable code the out of bounds read can only trigger a crash.\nFurthermore the vulnerability requires an attacker-controlled URL to be\npassed from an application to the OpenSSL function and the user has to have\na 'no_proxy' environment variable set. For the aforementioned reasons the\nissue was assessed as Low severity.\n\nThe vulnerable code was introduced in the following patch releases:\n3.0.16, 3.1.8, 3.2.4, 3.3.3, 3.4.0 and 3.5.0.\n\nThe FIPS modules in 3.5, 3.4, 3.3, 3.2, 3.1 and 3.0 are not affected by this\nissue, as the HTTP client implementation is outside the OpenSSL FIPS module\nboundary."

var nvdValue1335 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1334}

var nvdValue1336 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1335}, ID: "CVE-2025-9232", LastModified: "2026-07-14T13:18:07.783", Metrics: &nvdValue1233, Published: "2025-09-30T14:15:41.313"}

var nvdValue1337 = "The 'zipfile' module would not check the validity of the ZIP64 End of\nCentral Directory (EOCD) Locator record offset value would not be used to\nlocate the ZIP64 EOCD record, instead the ZIP64 EOCD record would be\nassumed to be the previous record in the ZIP archive. This could be abused\nto create ZIP archives that are handled differently by the 'zipfile' module\ncompared to other ZIP implementations.\n\n\nRemediation maintains this behavior, but checks that the offset specified\nin the ZIP64 EOCD Locator record matches the expected value."

var nvdValue1338 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1337}

var nvdValue1339 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1338}, ID: "CVE-2025-8291", LastModified: "2026-07-31T14:16:45.617", Metrics: &nvdValue1233, Published: "2025-10-07T18:16:00.317"}

var nvdValue1340 = "If the value passed to os.path.expandvars() is user-controlled a \nperformance degradation is possible when expanding environment \nvariables."

var nvdValue1341 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1340}

var nvdValue1342 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue427}}

var nvdValue1343 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1341}, ID: "CVE-2025-6075", LastModified: "2026-07-31T14:16:45.270", Metrics: &nvdValue1342, Published: "2025-10-31T17:15:48.693"}

var nvdValue1344 = "In libexpat through 2.7.3, a crafted file with an approximate size of 2 MiB can lead to dozens of seconds of processing time."

var nvdValue1345 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1344}

var nvdValue1346 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1345}, ID: "CVE-2025-66382", LastModified: "2026-06-17T09:56:45.240", Metrics: &nvdValue1237, Published: "2025-11-28T07:15:57.900"}

var nvdValue1347 = "Denial of Service vulnerability in Apache Struts, file leak in multipart request processing causes disk exhaustion.\n\nThis issue affects Apache Struts: from 2.0.0 through 6.7.0, from 7.0.0 through 7.0.3.\n\nUsers are recommended to upgrade to version 6.8.0 or 7.1.1, which fixes the issue."

var nvdValue1348 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1347}

var nvdValue1349 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1348}, ID: "CVE-2025-64775", LastModified: "2026-06-17T09:55:11.247", Metrics: &nvdValue1233, Published: "2025-12-01T16:15:56.873"}

var nvdValue1350 = "When loading a plist file, the plistlib module reads data in size specified by the file itself, meaning a malicious file can cause OOM and DoS issues"

var nvdValue1351 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1350}

var nvdValue1352 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1351}, ID: "CVE-2025-13837", LastModified: "2026-09-03T03:15:21.037", Metrics: &nvdValue1237, Published: "2025-12-01T18:16:04.380"}

var nvdValue1353 = "When building nested elements using xml.dom.minidom methods such as appendChild() that have a dependency on _clear_id_cache() the algorithm is quadratic. Availability can be impacted when building excessively nested documents."

var nvdValue1354 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1353}

var nvdValue1355 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1354}, ID: "CVE-2025-12084", LastModified: "2026-06-17T08:31:41.583", Metrics: &nvdValue1139, Published: "2025-12-03T19:15:55.050"}

var nvdValue1356 = "Denial of Service vulnerability in Apache Struts, file leak in multipart request processing causes disk exhaustion.\n\nThis issue affects Apache Struts: from 2.0.0 through 6.7.4, from 7.0.0 through 7.0.3.\n\nUsers are recommended to upgrade to version 6.8.0 or 7.1.1, which fixes the issue.\n\nIt's related to\u00a0 https://cve.org/CVERecord?id=CVE-2025-64775 \u00a0- this CVE addresses missing affected version 6.7.4"

var nvdValue1357 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1356}

var nvdValue1358 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1357}, ID: "CVE-2025-66675", LastModified: "2026-06-17T09:57:09.367", Metrics: &nvdValue1233, Published: "2025-12-10T10:16:02.170"}

var nvdValue1359 = "When doing multi-threaded LDAPS transfers (LDAP over TLS) with libcurl,\nchanging TLS options in one thread would inadvertently change them globally\nand therefore possibly also affect other concurrently setup transfers.\n\nDisabling certificate verification for a specific transfer could\nunintentionally disable the feature for other threads as well."

var nvdValue1360 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1359}

var nvdValue1361 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1360}, ID: "CVE-2025-14017", LastModified: "2026-09-15T07:16:23.753", Metrics: &nvdValue1233, Published: "2026-01-08T10:15:45.667"}

var nvdValue1362 = "When an OAuth2 bearer token is used for an HTTP(S) transfer, and that transfer\nperforms a cross-protocol redirect to a second URL that uses an IMAP, LDAP,\nPOP3 or SMTP scheme, curl might wrongly pass on the bearer token to the new\ntarget host."

var nvdValue1363 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1362}

var nvdValue1364 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1363}, ID: "CVE-2025-14524", LastModified: "2026-09-15T07:16:24.020", Metrics: &nvdValue1233, Published: "2026-01-08T10:15:46.607"}

var nvdValue1365 = "When doing SSH-based transfers using either SCP or SFTP, and setting the\nknown_hosts file, libcurl could still mistakenly accept connecting to hosts\n*not present* in the specified file if they were added as recognized in the\nlibssh *global* known_hosts file."

var nvdValue1366 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1365}

var nvdValue1367 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1366}, ID: "CVE-2025-15079", LastModified: "2026-09-15T07:16:24.467", Metrics: &nvdValue1233, Published: "2026-01-08T10:15:47.100"}

var nvdValue1368 = "When doing SSH-based transfers using either SCP or SFTP, and asked to do\npublic key authentication, curl would wrongly still ask and authenticate using\na locally running SSH agent."

var nvdValue1369 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1368}

var nvdValue1370 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1369}, ID: "CVE-2025-15224", LastModified: "2026-09-15T07:16:24.690", Metrics: &nvdValue1233, Published: "2026-01-08T10:15:47.207"}

var nvdValue1371 = "Missing XML Validation vulnerability in Apache Struts, Apache Struts.\n\nThis issue affects Apache Struts: from 2.0.0 before 2.2.1; Apache Struts: from 2.2.1 through 6.1.0.\n\nUsers are recommended to upgrade to version 6.1.1, which fixes the issue."

var nvdValue1372 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1371}

var nvdValue1373 = schema.CVSSV31{BaseScore: 8.1, VectorString: "CVSS:3.1/AV:N/AC:L/PR:N/UI:R/S:U/C:H/I:N/A:H", Version: "3.1"}

var nvdValue1374 = schema.CVEAPIJSON20CVSSV31{CvssData: &nvdValue1373}

var nvdValue1375 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue1374}}

var nvdValue1376 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1372}, ID: "CVE-2025-68493", LastModified: "2026-07-15T02:17:50.190", Metrics: &nvdValue1375, Published: "2026-01-11T13:15:45.610"}

var nvdValue1377 = "Calling wordexp with WRDE_REUSE in conjunction with WRDE_APPEND in the GNU C Library version 2.0 to version 2.42 may cause the interface to return uninitialized memory in the we_wordv member, which on subsequent calls to wordfree may abort the process."

var nvdValue1378 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1377}

var nvdValue1379 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1378}, ID: "CVE-2025-15281", LastModified: "2026-06-17T08:37:31.410", Metrics: &nvdValue1233, Published: "2026-01-20T14:16:07.843"}

var nvdValue1380 = "When folding a long comment in an email header containing exclusively unfoldable characters, the parenthesis would not be preserved. This could be used for injecting headers into email messages where addresses are user-controlled and not sanitized."

var nvdValue1381 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1380}

var nvdValue1382 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1381}, ID: "CVE-2025-11468", LastModified: "2026-06-17T08:30:31.043", Metrics: &nvdValue1233, Published: "2026-01-20T22:15:50.690"}

var nvdValue1383 = "User-controlled data URLs parsed by urllib.request.DataHandler allow injecting headers through newlines in the data URL mediatype."

var nvdValue1384 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1383}

var nvdValue1385 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1384}, ID: "CVE-2025-15282", LastModified: "2026-06-17T08:37:31.580", Metrics: &nvdValue1233, Published: "2026-01-20T22:15:50.883"}

var nvdValue1386 = "Issue summary: PBMAC1 parameters in PKCS#12 files are missing validation\nwhich can trigger a stack-based buffer overflow, invalid pointer or NULL\npointer dereference during MAC verification.\n\nImpact summary: The stack buffer overflow or NULL pointer dereference may\ncause a crash leading to Denial of Service for an application that parses\nuntrusted PKCS#12 files. The buffer overflow may also potentially enable\ncode execution depending on platform mitigations.\n\nWhen verifying a PKCS#12 file that uses PBMAC1 for the MAC, the PBKDF2\nsalt and keylength parameters from the file are used without validation.\nIf the value of keylength exceeds the size of the fixed stack buffer used\nfor the derived key (64 bytes), the key derivation will overflow the buffer.\nThe overflow length is attacker-controlled. Also, if the salt parameter is\nnot an OCTET STRING type this can lead to invalid or NULL pointer\ndereference.\n\nExploiting this issue requires a user or application to process\na maliciously crafted PKCS#12 file. It is uncommon to accept untrusted\nPKCS#12 files in applications as they are usually used to store private\nkeys which are trusted by definition. For this reason the issue was assessed\nas Moderate severity.\n\nThe FIPS modules in 3.6, 3.5 and 3.4 are not affected by this issue, as\nPKCS#12 processing is outside the OpenSSL FIPS module boundary.\n\nOpenSSL 3.6, 3.5 and 3.4 are vulnerable to this issue.\n\nOpenSSL 3.3, 3.0, 1.1.1 and 1.0.2 are not affected by this issue as they do\nnot support PBMAC1 in PKCS#12."

var nvdValue1387 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1386}

var nvdValue1388 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1387}, ID: "CVE-2025-11187", LastModified: "2026-06-17T08:29:49.083", Metrics: &nvdValue1233, Published: "2026-01-27T16:16:14.093"}

var nvdValue1389 = "Issue summary: Parsing CMS AuthEnvelopedData or EnvelopedData message with\nmaliciously crafted AEAD parameters can trigger a stack buffer overflow.\n\nImpact summary: A stack buffer overflow may lead to a crash, causing Denial\nof Service, or potentially remote code execution.\n\nWhen parsing CMS (Auth)EnvelopedData structures that use AEAD ciphers such as\nAES-GCM, the IV (Initialization Vector) encoded in the ASN.1 parameters is\ncopied into a fixed-size stack buffer without verifying that its length fits\nthe destination. An attacker can supply a crafted CMS message with an\noversized IV, causing a stack-based out-of-bounds write before any\nauthentication or tag verification occurs.\n\nApplications and services that parse untrusted CMS or PKCS#7 content using\nAEAD ciphers (e.g., S/MIME (Auth)EnvelopedData with AES-GCM) are vulnerable.\nBecause the overflow occurs prior to authentication, no valid key material\nis required to trigger it. While exploitability to remote code execution\ndepends on platform and toolchain mitigations, the stack-based write\nprimitive represents a severe risk.\n\nThe FIPS modules in 3.6, 3.5, 3.4, 3.3 and 3.0 are not affected by this\nissue, as the CMS implementation is outside the OpenSSL FIPS module\nboundary.\n\nOpenSSL 3.6, 3.5, 3.4, 3.3 and 3.0 are vulnerable to this issue.\n\nOpenSSL 1.1.1 and 1.0.2 are not affected by this issue."

var nvdValue1390 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1389}

var nvdValue1391 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1390}, ID: "CVE-2025-15467", LastModified: "2026-09-07T13:17:27.740", Metrics: &nvdValue1233, Published: "2026-01-27T16:16:14.257"}

var nvdValue1392 = "Issue summary: If an application using the SSL_CIPHER_find() function in\na QUIC protocol client or server receives an unknown cipher suite from\nthe peer, a NULL dereference occurs.\n\nImpact summary: A NULL pointer dereference leads to abnormal termination of\nthe running process causing Denial of Service.\n\nSome applications call SSL_CIPHER_find() from the client_hello_cb callback\non the cipher ID received from the peer. If this is done with an SSL object\nimplementing the QUIC protocol, NULL pointer dereference will happen if\nthe examined cipher ID is unknown or unsupported.\n\nAs it is not very common to call this function in applications using the QUIC \nprotocol and the worst outcome is Denial of Service, the issue was assessed\nas Low severity.\n\nThe vulnerable code was introduced in the 3.2 version with the addition\nof the QUIC protocol support.\n\nThe FIPS modules in 3.6, 3.5, 3.4 and 3.3 are not affected by this issue,\nas the QUIC implementation is outside the OpenSSL FIPS module boundary.\n\nOpenSSL 3.6, 3.5, 3.4 and 3.3 are vulnerable to this issue.\n\nOpenSSL 3.0, 1.1.1 and 1.0.2 are not affected by this issue."

var nvdValue1393 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1392}

var nvdValue1394 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1393}, ID: "CVE-2025-15468", LastModified: "2026-06-17T08:37:50.723", Metrics: &nvdValue1233, Published: "2026-01-27T16:16:14.400"}

var nvdValue1395 = "Issue summary: The 'openssl dgst' command-line tool silently truncates input\ndata to 16MB when using one-shot signing algorithms and reports success instead\nof an error.\n\nImpact summary: A user signing or verifying files larger than 16MB with\none-shot algorithms (such as Ed25519, Ed448, or ML-DSA) may believe the entire\nfile is authenticated while trailing data beyond 16MB remains unauthenticated.\n\nWhen the 'openssl dgst' command is used with algorithms that only support\none-shot signing (Ed25519, Ed448, ML-DSA-44, ML-DSA-65, ML-DSA-87), the input\nis buffered with a 16MB limit. If the input exceeds this limit, the tool\nsilently truncates to the first 16MB and continues without signaling an error,\ncontrary to what the documentation states. This creates an integrity gap where\ntrailing bytes can be modified without detection if both signing and\nverification are performed using the same affected codepath.\n\nThe issue affects only the command-line tool behavior. Verifiers that process\nthe full message using library APIs will reject the signature, so the risk\nprimarily affects workflows that both sign and verify with the affected\n'openssl dgst' command. Streaming digest algorithms for 'openssl dgst' and\nlibrary users are unaffected.\n\nThe FIPS modules in 3.5 and 3.6 are not affected by this issue, as the\ncommand-line tools are outside the OpenSSL FIPS module boundary.\n\nOpenSSL 3.5 and 3.6 are vulnerable to this issue.\n\nOpenSSL 3.4, 3.3, 3.0, 1.1.1 and 1.0.2 are not affected by this issue."

var nvdValue1396 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1395}

var nvdValue1397 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1396}, ID: "CVE-2025-15469", LastModified: "2026-06-17T08:37:50.893", Metrics: &nvdValue1233, Published: "2026-01-27T16:16:14.523"}

var nvdValue1398 = "Issue summary: A TLS 1.3 connection using certificate compression can be\nforced to allocate a large buffer before decompression without checking\nagainst the configured certificate size limit.\n\nImpact summary: An attacker can cause per-connection memory allocations of\nup to approximately 22 MiB and extra CPU work, potentially leading to\nservice degradation or resource exhaustion (Denial of Service).\n\nIn affected configurations, the peer-supplied uncompressed certificate\nlength from a CompressedCertificate message is used to grow a heap buffer\nprior to decompression. This length is not bounded by the max_cert_list\nsetting, which otherwise constrains certificate message sizes. An attacker\ncan exploit this to cause large per-connection allocations followed by\nhandshake failure. No memory corruption or information disclosure occurs.\n\nThis issue only affects builds where TLS 1.3 certificate compression is\ncompiled in (i.e., not OPENSSL_NO_COMP_ALG) and at least one compression\nalgorithm (brotli, zlib, or zstd) is available, and where the compression\nextension is negotiated. Both clients receiving a server CompressedCertificate\nand servers in mutual TLS scenarios receiving a client CompressedCertificate\nare affected. Servers that do not request client certificates are not\nvulnerable to client-initiated attacks.\n\nUsers can mitigate this issue by setting SSL_OP_NO_RX_CERTIFICATE_COMPRESSION\nto disable receiving compressed certificates.\n\nThe FIPS modules in 3.6, 3.5, 3.4 and 3.3 are not affected by this issue,\nas the TLS implementation is outside the OpenSSL FIPS module boundary.\n\nOpenSSL 3.6, 3.5, 3.4 and 3.3 are vulnerable to this issue.\n\nOpenSSL 3.0, 1.1.1 and 1.0.2 are not affected by this issue."

var nvdValue1399 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1398}

var nvdValue1400 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1399}, ID: "CVE-2025-66199", LastModified: "2026-06-17T09:56:27.733", Metrics: &nvdValue1233, Published: "2026-01-27T16:16:15.777"}

var nvdValue1401 = "Issue summary: Writing large, newline-free data into a BIO chain using the\nline-buffering filter where the next BIO performs short writes can trigger\na heap-based out-of-bounds write.\n\nImpact summary: This out-of-bounds write can cause memory corruption which\ntypically results in a crash, leading to Denial of Service for an application.\n\nThe line-buffering BIO filter (BIO_f_linebuffer) is not used by default in\nTLS/SSL data paths. In OpenSSL command-line applications, it is typically\nonly pushed onto stdout/stderr on VMS systems. Third-party applications that\nexplicitly use this filter with a BIO chain that can short-write and that\nwrite large, newline-free data influenced by an attacker would be affected.\nHowever, the circumstances where this could happen are unlikely to be under\nattacker control, and BIO_f_linebuffer is unlikely to be handling non-curated\ndata controlled by an attacker. For that reason the issue was assessed as\nLow severity.\n\nThe FIPS modules in 3.6, 3.5, 3.4, 3.3 and 3.0 are not affected by this issue,\nas the BIO implementation is outside the OpenSSL FIPS module boundary.\n\nOpenSSL 3.6, 3.5, 3.4, 3.3, 3.0, 1.1.1 and 1.0.2 are vulnerable to this issue."

var nvdValue1402 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1401}

var nvdValue1403 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1402}, ID: "CVE-2025-68160", LastModified: "2026-06-17T09:58:39.407", Metrics: &nvdValue1233, Published: "2026-01-27T16:16:15.900"}

var nvdValue1404 = "Issue summary: When using the low-level OCB API directly with AES-NI or<br>other hardware-accelerated code paths, inputs whose length is not a multiple<br>of 16 bytes can leave the final partial block unencrypted and unauthenticated.<br><br>Impact summary: The trailing 1-15 bytes of a message may be exposed in<br>cleartext on encryption and are not covered by the authentication tag,<br>allowing an attacker to read or tamper with those bytes without detection.<br><br>The low-level OCB encrypt and decrypt routines in the hardware-accelerated<br>stream path process full 16-byte blocks but do not advance the input/output<br>pointers. The subsequent tail-handling code then operates on the original<br>base pointers, effectively reprocessing the beginning of the buffer while<br>leaving the actual trailing bytes unprocessed. The authentication checksum<br>also excludes the true tail bytes.<br><br>However, typical OpenSSL consumers using EVP are not affected because the<br>higher-level EVP and provider OCB implementations split inputs so that full<br>blocks and trailing partial blocks are processed in separate calls, avoiding<br>the problematic code path. Additionally, TLS does not use OCB ciphersuites.<br>The vulnerability only affects applications that call the low-level<br>CRYPTO_ocb128_encrypt() or CRYPTO_ocb128_decrypt() functions directly with<br>non-block-aligned lengths in a single call on hardware-accelerated builds.<br>For these reasons the issue was assessed as Low severity.<br><br>The FIPS modules in 3.6, 3.5, 3.4, 3.3, 3.2, 3.1 and 3.0 are not affected<br>by this issue, as OCB mode is not a FIPS-approved algorithm.<br><br>OpenSSL 3.6, 3.5, 3.4, 3.3, 3.0 and 1.1.1 are vulnerable to this issue.<br><br>OpenSSL 1.0.2 is not affected by this issue."

var nvdValue1405 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1404}

var nvdValue1406 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1405}, ID: "CVE-2025-69418", LastModified: "2026-06-17T10:00:39.647", Metrics: &nvdValue1233, Published: "2026-01-27T16:16:33.253"}

var nvdValue1407 = "Issue summary: Calling PKCS12_get_friendlyname() function on a maliciously\ncrafted PKCS#12 file with a BMPString (UTF-16BE) friendly name containing\nnon-ASCII BMP code point can trigger a one byte write before the allocated\nbuffer.\n\nImpact summary: The out-of-bounds write can cause a memory corruption\nwhich can have various consequences including a Denial of Service.\n\nThe OPENSSL_uni2utf8() function performs a two-pass conversion of a PKCS#12\nBMPString (UTF-16BE) to UTF-8. In the second pass, when emitting UTF-8 bytes,\nthe helper function bmp_to_utf8() incorrectly forwards the remaining UTF-16\nsource byte count as the destination buffer capacity to UTF8_putc(). For BMP\ncode points above U+07FF, UTF-8 requires three bytes, but the forwarded\ncapacity can be just two bytes. UTF8_putc() then returns -1, and this negative\nvalue is added to the output length without validation, causing the\nlength to become negative. The subsequent trailing NUL byte is then written\nat a negative offset, causing write outside of heap allocated buffer.\n\nThe vulnerability is reachable via the public PKCS12_get_friendlyname() API\nwhen parsing attacker-controlled PKCS#12 files. While PKCS12_parse() uses a\ndifferent code path that avoids this issue, PKCS12_get_friendlyname() directly\ninvokes the vulnerable function. Exploitation requires an attacker to provide\na malicious PKCS#12 file to be parsed by the application and the attacker\ncan just trigger a one zero byte write before the allocated buffer.\nFor that reason the issue was assessed as Low severity according to our\nSecurity Policy.\n\nThe FIPS modules in 3.6, 3.5, 3.4, 3.3 and 3.0 are not affected by this issue,\nas the PKCS#12 implementation is outside the OpenSSL FIPS module boundary.\n\nOpenSSL 3.6, 3.5, 3.4, 3.3, 3.0 and 1.1.1 are vulnerable to this issue.\n\nOpenSSL 1.0.2 is not affected by this issue."

var nvdValue1408 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1407}

var nvdValue1409 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1408}, ID: "CVE-2025-69419", LastModified: "2026-06-17T10:00:39.850", Metrics: &nvdValue1233, Published: "2026-01-27T16:16:34.113"}

var nvdValue1410 = "Issue summary: A type confusion vulnerability exists in the TimeStamp Response\nverification code where an ASN1_TYPE union member is accessed without first\nvalidating the type, causing an invalid or NULL pointer dereference when\nprocessing a malformed TimeStamp Response file.\n\nImpact summary: An application calling TS_RESP_verify_response() with a\nmalformed TimeStamp Response can be caused to dereference an invalid or\nNULL pointer when reading, resulting in a Denial of Service.\n\nThe functions ossl_ess_get_signing_cert() and ossl_ess_get_signing_cert_v2()\naccess the signing cert attribute value without validating its type.\nWhen the type is not V_ASN1_SEQUENCE, this results in accessing invalid memory\nthrough the ASN1_TYPE union, causing a crash.\n\nExploiting this vulnerability requires an attacker to provide a malformed\nTimeStamp Response to an application that verifies timestamp responses. The\nTimeStamp protocol (RFC 3161) is not widely used and the impact of the\nexploit is just a Denial of Service. For these reasons the issue was\nassessed as Low severity.\n\nThe FIPS modules in 3.5, 3.4, 3.3 and 3.0 are not affected by this issue,\nas the TimeStamp Response implementation is outside the OpenSSL FIPS module\nboundary.\n\nOpenSSL 3.6, 3.5, 3.4, 3.3, 3.0 and 1.1.1 are vulnerable to this issue.\n\nOpenSSL 1.0.2 is not affected by this issue."

var nvdValue1411 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1410}

var nvdValue1412 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1411}, ID: "CVE-2025-69420", LastModified: "2026-06-17T10:00:40.067", Metrics: &nvdValue1233, Published: "2026-01-27T16:16:34.317"}

var nvdValue1413 = "Issue summary: Processing a malformed PKCS#12 file can trigger a NULL pointer\ndereference in the PKCS12_item_decrypt_d2i_ex() function.\n\nImpact summary: A NULL pointer dereference can trigger a crash which leads to\nDenial of Service for an application processing PKCS#12 files.\n\nThe PKCS12_item_decrypt_d2i_ex() function does not check whether the oct\nparameter is NULL before dereferencing it. When called from\nPKCS12_unpack_p7encdata() with a malformed PKCS#12 file, this parameter can\nbe NULL, causing a crash. The vulnerability is limited to Denial of Service\nand cannot be escalated to achieve code execution or memory disclosure.\n\nExploiting this issue requires an attacker to provide a malformed PKCS#12 file\nto an application that processes it. For that reason the issue was assessed as\nLow severity according to our Security Policy.\n\nThe FIPS modules in 3.6, 3.5, 3.4, 3.3 and 3.0 are not affected by this issue,\nas the PKCS#12 implementation is outside the OpenSSL FIPS module boundary.\n\nOpenSSL 3.6, 3.5, 3.4, 3.3, 3.0, 1.1.1 and 1.0.2 are vulnerable to this issue."

var nvdValue1414 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1413}

var nvdValue1415 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1414}, ID: "CVE-2025-69421", LastModified: "2026-06-17T10:00:40.683", Metrics: &nvdValue997, Published: "2026-01-27T16:16:34.437"}

var nvdValue1416 = "Passing too large an alignment to the memalign suite of functions (memalign, posix_memalign, aligned_alloc) in the GNU C Library version 2.30 to 2.42 may result in an integer overflow, which could consequently result in a heap corruption.\n\nNote that the attacker must have control over both, the size as well as the alignment arguments of the memalign function to be able to exploit this.  The size parameter must be close enough to PTRDIFF_MAX so as to overflow size_t along with the large alignment argument.  This limits the malicious inputs for the alignment for memalign to the range [1<<62+ 1, 1<<63] and exactly 1<<63 for posix_memalign and aligned_alloc.\n\nTypically the alignment argument passed to such functions is a known constrained quantity (e.g. page size, block size, struct sizes) and is not attacker controlled, because of which this may not be easily exploitable in practice.  An application bug could potentially result in the input alignment being too large, e.g. due to a different buffer overflow or integer overflow in the application or its dependent libraries, but that is again an uncommon usage pattern given typical sources of alignments."

var nvdValue1417 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1416}

var nvdValue1418 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1417}, ID: "CVE-2026-0861", LastModified: "2026-06-17T10:11:31.167", Metrics: &nvdValue1233, Published: "2026-01-14T21:15:52.617"}

var nvdValue1419 = "Calling getnetbyaddr or getnetbyaddr_r with a configured nsswitch.conf that specifies the library's DNS backend for networks and queries for a zero-valued network in the GNU C Library version 2.0 to version 2.42 can leak stack contents to the configured DNS resolver."

var nvdValue1420 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1419}

var nvdValue1421 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1420}, ID: "CVE-2026-0915", LastModified: "2026-06-17T10:11:37.513", Metrics: &nvdValue1233, Published: "2026-01-15T22:16:12.457"}

var nvdValue1422 = "When using http.cookies.Morsel, user-controlled cookie values and parameters can allow injecting HTTP headers into messages. Patch rejects all control characters within cookie names, values, and parameters."

var nvdValue1423 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1422}

var nvdValue1424 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1423}, ID: "CVE-2026-0672", LastModified: "2026-06-17T10:11:11.133", Metrics: &nvdValue1233, Published: "2026-01-20T22:15:52.680"}

var nvdValue1425 = schema.CVEAPIJSON20LangString{Lang: "en", Value: "User-controlled header names and values containing newlines can allow injecting HTTP headers."}

var nvdValue1426 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1425}, ID: "CVE-2026-0865", LastModified: "2026-06-17T10:11:31.550", Metrics: &nvdValue1233, Published: "2026-01-20T22:15:52.800"}

var nvdValue1427 = schema.CVEAPIJSON20LangString{Lang: "en", Value: "In libexpat before 2.7.4, XML_ExternalEntityParserCreate does not copy unknown encoding handler user data."}

var nvdValue1428 = schema.CVSSV31{BaseScore: 2.5, VectorString: "CVSS:3.1/AV:L/AC:H/PR:L/UI:N/S:U/C:N/I:N/A:L", Version: "3.1"}

var nvdValue1429 = schema.CVEAPIJSON20CVSSV31{CvssData: &nvdValue1428}

var nvdValue1430 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue1429}}

var nvdValue1431 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1427}, ID: "CVE-2026-24515", LastModified: "2026-06-17T10:23:10.660", Metrics: &nvdValue1430, Published: "2026-01-23T08:16:01.490"}

var nvdValue1432 = "The \nemail module, specifically the \"BytesGenerator\" class, didn’t properly quote newlines for email headers when \nserializing an email message allowing for header injection when an email\n is serialized. This is only applicable if using \"LiteralHeader\" writing headers that don't respect email folding rules, the new behavior will reject the incorrectly folded headers in \"BytesGenerator\"."

var nvdValue1433 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1432}

var nvdValue1434 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1433}, ID: "CVE-2026-1299", LastModified: "2026-06-17T10:15:28.543", Metrics: &nvdValue1233, Published: "2026-01-23T17:16:12.977"}

var nvdValue1435 = "Issue summary: An invalid or NULL pointer dereference can happen in\nan application processing a malformed PKCS#12 file.\n\nImpact summary: An application processing a malformed PKCS#12 file can be\ncaused to dereference an invalid or NULL pointer on memory read, resulting\nin a Denial of Service.\n\nA type confusion vulnerability exists in PKCS#12 parsing code where\nan ASN1_TYPE union member is accessed without first validating the type,\ncausing an invalid pointer read.\n\nThe location is constrained to a 1-byte address space, meaning any\nattempted pointer manipulation can only target addresses between 0x00 and 0xFF.\nThis range corresponds to the zero page, which is unmapped on most modern\noperating systems and will reliably result in a crash, leading only to a\nDenial of Service. Exploiting this issue also requires a user or application\nto process a maliciously crafted PKCS#12 file. It is uncommon to accept\nuntrusted PKCS#12 files in applications as they are usually used to store\nprivate keys which are trusted by definition. For these reasons, the issue\nwas assessed as Low severity.\n\nThe FIPS modules in 3.5, 3.4, 3.3 and 3.0 are not affected by this issue,\nas the PKCS12 implementation is outside the OpenSSL FIPS module boundary.\n\nOpenSSL 3.6, 3.5, 3.4, 3.3, 3.0 and 1.1.1 are vulnerable to this issue.\n\nOpenSSL 1.0.2 is not affected by this issue."

var nvdValue1436 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1435}

var nvdValue1437 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1436}, ID: "CVE-2026-22795", LastModified: "2026-06-17T10:20:26.520", Metrics: &nvdValue1233, Published: "2026-01-27T16:16:35.430"}

var nvdValue1438 = "Issue summary: A type confusion vulnerability exists in the signature\nverification of signed PKCS#7 data where an ASN1_TYPE union member is\naccessed without first validating the type, causing an invalid or NULL\npointer dereference when processing malformed PKCS#7 data.\n\nImpact summary: An application performing signature verification of PKCS#7\ndata or calling directly the PKCS7_digest_from_attributes() function can be\ncaused to dereference an invalid or NULL pointer when reading, resulting in\na Denial of Service.\n\nThe function PKCS7_digest_from_attributes() accesses the message digest attribute\nvalue without validating its type. When the type is not V_ASN1_OCTET_STRING,\nthis results in accessing invalid memory through the ASN1_TYPE union, causing\na crash.\n\nExploiting this vulnerability requires an attacker to provide a malformed\nsigned PKCS#7 to an application that verifies it. The impact of the\nexploit is just a Denial of Service, the PKCS7 API is legacy and applications\nshould be using the CMS API instead. For these reasons the issue was\nassessed as Low severity.\n\nThe FIPS modules in 3.5, 3.4, 3.3 and 3.0 are not affected by this issue,\nas the PKCS#7 parsing implementation is outside the OpenSSL FIPS module\nboundary.\n\nOpenSSL 3.6, 3.5, 3.4, 3.3, 3.0, 1.1.1 and 1.0.2 are vulnerable to this issue."

var nvdValue1439 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1438}

var nvdValue1440 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1439}, ID: "CVE-2026-22796", LastModified: "2026-06-17T10:20:26.697", Metrics: &nvdValue1233, Published: "2026-01-27T16:16:35.543"}

var nvdValue1441 = "In libexpat before 2.7.4, the doContent function does not properly determine the buffer size bufSize because there is no integer overflow check for tag buffer reallocation."

var nvdValue1442 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1441}

var nvdValue1443 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1442}, ID: "CVE-2026-25210", LastModified: "2026-06-17T10:24:19.123", Metrics: &nvdValue1173, Published: "2026-01-30T07:16:15.570"}

var nvdValue1444 = "The import hook in CPython that handles legacy *.pyc files (SourcelessFileLoader) is incorrectly handled in FileLoader (a base class) and so does not use io.open_code() to read the .pyc files. sys.audit handlers for this audit event therefore do not fire."

var nvdValue1445 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1444}

var nvdValue1446 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1445}, ID: "CVE-2026-2297", LastModified: "2026-08-13T01:16:52.527", Metrics: &nvdValue1233, Published: "2026-03-04T23:16:10.757"}

var nvdValue1447 = "libcurl can in some circumstances reuse the wrong connection when asked to do\nan Negotiate-authenticated HTTP or HTTPS request.\n\nlibcurl features a pool of recent connections so that subsequent requests can\nreuse an existing connection to avoid overhead.\n\nWhen reusing a connection a range of criterion must first be met. Due to a\nlogical error in the code, a request that was issued by an application could\nwrongfully reuse an existing connection to the same server that was\nauthenticated using different credentials. One underlying reason being that\nNegotiate sometimes authenticates *connections* and not *requests*, contrary\nto how HTTP is designed to work.\n\nAn application that allows Negotiate authentication to a server (that responds\nwanting Negotiate) with `user1:password1` and then does another operation to\nthe same server also using Negotiate but with `user2:password2` (while the\nprevious connection is still alive) - the second request wrongly reused the\nsame connection and since it then sees that the Negotiate negotiation is\nalready made, it sends the request over that connection thinking it uses\nthe user2 credentials when it is in fact still using the connection\nauthenticated for user1...\n\nThe set of authentication methods to use is set with `CURLOPT_HTTPAUTH`.\n\nApplications can disable libcurl's reuse of connections and thus mitigate this\nproblem, by using one of the following libcurl options to alter how\nconnections are or are not reused: `CURLOPT_FRESH_CONNECT`,\n`CURLOPT_MAXCONNECTS` and `CURLMOPT_MAX_HOST_CONNECTIONS` (if using the\ncurl_multi API)."

var nvdValue1448 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1447}

var nvdValue1449 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1448}, ID: "CVE-2026-1965", LastModified: "2026-09-15T07:16:27.523", Metrics: &nvdValue1233, Published: "2026-03-11T11:15:59.177"}

var nvdValue1450 = "When an OAuth2 bearer token is used for an HTTP(S) transfer, and that transfer\nperforms a redirect to a second URL, curl could leak that token to the second\nhostname under some circumstances.\n\nIf the hostname that the first request is redirected to has information in the\nused .netrc file, with either of the `machine` or `default` keywords, curl\nwould pass on the bearer token set for the first host also to the second one."

var nvdValue1451 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1450}

var nvdValue1452 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1451}, ID: "CVE-2026-3783", LastModified: "2026-09-15T07:16:27.720", Metrics: &nvdValue1233, Published: "2026-03-11T11:16:00.080"}

var nvdValue1453 = "curl would wrongly reuse an existing HTTP proxy connection doing CONNECT to a\nserver, even if the new request uses different credentials for the HTTP proxy.\nThe proper behavior is to create or use a separate connection."

var nvdValue1454 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1453}

var nvdValue1455 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1454}, ID: "CVE-2026-3784", LastModified: "2026-09-15T07:16:27.963", Metrics: &nvdValue1233, Published: "2026-03-11T11:16:00.437"}

var nvdValue1456 = schema.CVEAPIJSON20LangString{Lang: "en", Value: "libexpat before 2.7.5 allows a NULL pointer dereference with empty external parameter entity content."}

var nvdValue1457 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1456}, ID: "CVE-2026-32776", LastModified: "2026-07-14T13:18:49.530", Metrics: &nvdValue1342, Published: "2026-03-16T14:19:44.600"}

var nvdValue1458 = schema.CVEAPIJSON20LangString{Lang: "en", Value: "libexpat before 2.7.5 allows an infinite loop while parsing DTD content."}

var nvdValue1459 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1458}, ID: "CVE-2026-32777", LastModified: "2026-07-14T13:18:49.687", Metrics: &nvdValue1342, Published: "2026-03-16T14:19:44.780"}

var nvdValue1460 = "libexpat before 2.7.5 allows a NULL pointer dereference in the function setContext on retry after an earlier ouf-of-memory condition."

var nvdValue1461 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1460}

var nvdValue1462 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1461}, ID: "CVE-2026-32778", LastModified: "2026-07-14T13:18:49.843", Metrics: &nvdValue1342, Published: "2026-03-16T14:19:44.970"}

var nvdValue1463 = "The fix for CVE-2026-0672, which rejected control characters in http.cookies.Morsel, was incomplete. The Morsel.update(), |= operator, and unpickling paths were not patched, allowing control characters to bypass input validation. Additionally, BaseCookie.js_output() lacked the output validation applied to BaseCookie.output()."

var nvdValue1464 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1463}

var nvdValue1465 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1464}, ID: "CVE-2026-3644", LastModified: "2026-08-13T01:16:53.120", Metrics: &nvdValue1132, Published: "2026-03-16T18:16:09.907"}

var nvdValue1466 = "When an Expat parser with a registered ElementDeclHandler parses an inline\ndocument type definition containing a deeply nested content model a C stack\noverflow occurs."

var nvdValue1467 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1466}

var nvdValue1468 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1467}, ID: "CVE-2026-4224", LastModified: "2026-08-13T01:16:53.273", Metrics: &nvdValue997, Published: "2026-03-16T18:16:10.070"}

var nvdValue1469 = "The webbrowser.open() API would accept leading dashes in the URL which \ncould be handled as command line options for certain web browsers. New \nbehavior rejects leading dashes. Users are recommended to sanitize URLs \nprior to passing to webbrowser.open()."

var nvdValue1470 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1469}

var nvdValue1471 = schema.CVSSV31{BaseScore: 3.3, VectorString: "CVSS:3.1/AV:L/AC:L/PR:N/UI:R/S:U/C:N/I:L/A:N", Version: "3.1"}

var nvdValue1472 = schema.CVEAPIJSON20CVSSV31{CvssData: &nvdValue1471}

var nvdValue1473 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue1472}}

var nvdValue1474 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1470}, ID: "CVE-2026-4519", LastModified: "2026-08-13T01:16:53.580", Metrics: &nvdValue1473, Published: "2026-03-20T15:16:24.057"}

var nvdValue1475 = "Calling gethostbyaddr or gethostbyaddr_r with a configured nsswitch.conf that specifies the library's DNS backend in the GNU C Library version 2.34 to version 2.43 could, with a crafted response from the configured DNS server, result in a violation of the DNS specification that causes the application to treat a non-answer section of the DNS response as a valid answer."

var nvdValue1476 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1475}

var nvdValue1477 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1476}, ID: "CVE-2026-4437", LastModified: "2026-07-14T13:18:57.923", Metrics: &nvdValue1233, Published: "2026-03-20T20:16:49.477"}

var nvdValue1478 = "The iconv() function in the GNU C Library versions 2.43 and earlier may crash due to an assertion failure when converting inputs from the IBM1390 or IBM1399 character sets, which may be used to remotely crash an application.\n\n\n\nThis vulnerability can be trivially mitigated by removing the IBM1390 and IBM1399 character sets from systems that do not need them."

var nvdValue1479 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1478}

var nvdValue1480 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1479}, ID: "CVE-2026-4046", LastModified: "2026-07-14T13:18:57.707", Metrics: &nvdValue1233, Published: "2026-03-30T18:16:19.573"}

var nvdValue1481 = "Issue summary: An uncommon configuration of clients performing DANE TLSA-based\nserver authentication, when paired with uncommon server DANE TLSA records, may\nresult in a use-after-free and/or double-free on the client side.\n\nImpact summary: A use after free can have a range of potential consequences\nsuch as the corruption of valid data, crashes or execution of arbitrary code.\n\nHowever, the issue only affects clients that make use of TLSA records with both\nthe PKIX-TA(0/PKIX-EE(1) certificate usages and the DANE-TA(2) certificate\nusage.\n\nBy far the most common deployment of DANE is in SMTP MTAs for which RFC7672\nrecommends that clients treat as 'unusable' any TLSA records that have the PKIX\ncertificate usages.  These SMTP (or other similar) clients are not vulnerable\nto this issue.  Conversely, any clients that support only the PKIX usages, and\nignore the DANE-TA(2) usage are also not vulnerable.\n\nThe client would also need to be communicating with a server that publishes a\nTLSA RRset with both types of TLSA records.\n\nNo FIPS modules are affected by this issue, the problem code is outside the\nFIPS module boundary."

var nvdValue1482 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1481}

var nvdValue1483 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1482}, ID: "CVE-2026-28387", LastModified: "2026-07-24T23:10:00.563", Metrics: &nvdValue963, Published: "2026-04-07T22:16:20.700"}

var nvdValue1484 = "Issue summary: When a delta CRL that contains a Delta CRL Indicator extension\nis processed a NULL pointer dereference might happen if the required CRL\nNumber extension is missing.\n\nImpact summary: A NULL pointer dereference can trigger a crash which\nleads to a Denial of Service for an application.\n\nWhen CRL processing and delta CRL processing is enabled during X.509\ncertificate verification, the delta CRL processing does not check\nwhether the CRL Number extension is NULL before dereferencing it.\nWhen a malformed delta CRL file is being processed, this parameter\ncan be NULL, causing a NULL pointer dereference.\n\nExploiting this issue requires the X509_V_FLAG_USE_DELTAS flag to be enabled in\nthe verification context, the certificate being verified to contain a\nfreshestCRL extension or the base CRL to have the EXFLAG_FRESHEST flag set, and\nan attacker to provide a malformed CRL to an application that processes it.\n\nThe vulnerability is limited to Denial of Service and cannot be escalated to\nachieve code execution or memory disclosure. For that reason the issue was\nassessed as Low severity according to our Security Policy.\n\nThe FIPS modules in 3.6, 3.5, 3.4, 3.3 and 3.0 are not affected by this issue,\nas the affected code is outside the OpenSSL FIPS module boundary."

var nvdValue1485 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1484}

var nvdValue1486 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1485}, ID: "CVE-2026-28388", LastModified: "2026-07-24T23:10:00.563", Metrics: &nvdValue997, Published: "2026-04-07T22:16:20.863"}

var nvdValue1487 = "Issue summary: During processing of a crafted CMS EnvelopedData message\nwith KeyAgreeRecipientInfo a NULL pointer dereference can happen.\n\nImpact summary: Applications that process attacker-controlled CMS data may\ncrash before authentication or cryptographic operations occur resulting in\nDenial of Service.\n\nWhen a CMS EnvelopedData message that uses KeyAgreeRecipientInfo is\nprocessed, the optional parameters field of KeyEncryptionAlgorithmIdentifier\nis examined without checking for its presence. This results in a NULL\npointer dereference if the field is missing.\n\nApplications and services that call CMS_decrypt() on untrusted input\n(e.g., S/MIME processing or CMS-based protocols) are vulnerable.\n\nThe FIPS modules in 3.6, 3.5, 3.4, 3.3 and 3.0 are not affected by this\nissue, as the affected code is outside the OpenSSL FIPS module boundary."

var nvdValue1488 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1487}

var nvdValue1489 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1488}, ID: "CVE-2026-28389", LastModified: "2026-07-24T23:10:00.563", Metrics: &nvdValue997, Published: "2026-04-07T22:16:21.030"}

var nvdValue1490 = "Issue summary: During processing of a crafted CMS EnvelopedData message\nwith KeyTransportRecipientInfo a NULL pointer dereference can happen.\n\nImpact summary: Applications that process attacker-controlled CMS data may\ncrash before authentication or cryptographic operations occur resulting in\nDenial of Service.\n\nWhen a CMS EnvelopedData message that uses KeyTransportRecipientInfo with\nRSA-OAEP encryption is processed, the optional parameters field of\nRSA-OAEP SourceFunc algorithm identifier is examined without checking\nfor its presence. This results in a NULL pointer dereference if the field\nis missing.\n\nApplications and services that call CMS_decrypt() on untrusted input\n(e.g., S/MIME processing or CMS-based protocols) are vulnerable.\n\nThe FIPS modules in 3.6, 3.5, 3.4, 3.3 and 3.0 are not affected by this\nissue, as the affected code is outside the OpenSSL FIPS module boundary."

var nvdValue1491 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1490}

var nvdValue1492 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1491}, ID: "CVE-2026-28390", LastModified: "2026-07-24T23:10:00.563", Metrics: &nvdValue997, Published: "2026-04-07T22:16:21.190"}

var nvdValue1493 = "Mitgation of\u00a0CVE-2026-4519 was incomplete. If the URL contained \"%action\" the mitigation could be bypassed for certain browser types the \"webbrowser.open()\" API could have commands injected into the underlying shell. See\u00a0CVE-2026-4519 for details."

var nvdValue1494 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1493}

var nvdValue1495 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1494}, ID: "CVE-2026-4786", LastModified: "2026-08-13T01:16:54.237", Metrics: &nvdValue1233, Published: "2026-04-13T22:16:30.413"}

var nvdValue1496 = schema.CVEAPIJSON20LangString{Lang: "en", Value: "libexpat before 2.8.0 uses insufficient entropy, and thus hash flooding can occur via a crafted XML document."}

var nvdValue1497 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1496}, ID: "CVE-2026-41080", LastModified: "2026-07-14T13:18:51.257", Metrics: &nvdValue1233, Published: "2026-04-16T17:16:54.917"}

var nvdValue1498 = "In libexpat before 2.8.1, the computational complexity of attribute name collision checks allows a denial of service via moderately sized crafted XML input."

var nvdValue1499 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1498}

var nvdValue1500 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1499}, ID: "CVE-2026-45186", LastModified: "2026-09-16T13:17:58.967", Metrics: &nvdValue997, Published: "2026-05-10T07:16:07.883"}

var nvdValue1501 = "A vulnerability exists where a connection requiring TLS incorrectly reuses an\nexisting unencrypted connection from the same connection pool. If an initial\ntransfer is made in clear-text (via IMAP, SMTP, or POP3), a subsequent request\nto that same host bypasses the TLS requirement and instead transmit data\nunencrypted."

var nvdValue1502 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1501}

var nvdValue1503 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1502}, ID: "CVE-2026-4873", LastModified: "2026-09-15T07:16:28.367", Metrics: &nvdValue1233, Published: "2026-05-13T13:01:55.893"}

var nvdValue1504 = "libcurl might in some circumstances reuse the wrong connection when asked to\ndo an authenticated HTTP(S) request after a Negotiate-authenticated one, when\nboth use the same host.\n\nlibcurl features a pool of recent connections so that subsequent requests can\nreuse an existing connection to avoid overhead.\n\nWhen reusing a connection a range of criteria must be met. Due to a logical\nerror in the code, a request that was issued by an application could\nwrongfully reuse an existing connection to the same server that was\nauthenticated using different credentials.\n\nAn application that first uses Negotiate authentication to a server with\n`user1:password1` and then does another operation to the same server asking\nfor any authentication method but for `user2:password2` (while the previous\nconnection is still alive) - the second request gets confused and wrongly\nreuses the same connection and sends the new request over that connection\nthinking it uses a mix of user1's and user2's credentials when it is in fact\nstill using the connection authenticated for user1..."

var nvdValue1505 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1504}

var nvdValue1506 = schema.CVSSV31{BaseScore: 6.5, VectorString: "CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:L/I:H/A:N", Version: "3.1"}

var nvdValue1507 = schema.CVEAPIJSON20CVSSV31{CvssData: &nvdValue1506}

var nvdValue1508 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue1507}}

var nvdValue1509 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1505}, ID: "CVE-2026-5545", LastModified: "2026-09-15T07:16:28.633", Metrics: &nvdValue1508, Published: "2026-05-13T13:01:56.190"}

var nvdValue1510 = "libcurl might in some circumstances reuse the wrong connection for SMB(S)\ntransfers.\n\nlibcurl features a pool of recent connections so that subsequent requests can\nreuse an existing connection to avoid overhead.\n\nWhen reusing a connection a range of criteria must be met. Due to a logical\nerror in the code, a network transfer operation that was requested by an\napplication could wrongfully reuse an existing SMB connection to the same\nserver that was using a different \"share\" than the new subsequent transfer\nshould.\n\nThis could in unlucky situations lead to the download of the wrong file or the\nupload of a file to the wrong place. When this happens, the same credentials\nare used and the server name is the same."

var nvdValue1511 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1510}

var nvdValue1512 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1511}, ID: "CVE-2026-5773", LastModified: "2026-09-15T07:16:28.820", Metrics: &nvdValue993, Published: "2026-05-13T13:01:56.307"}

var nvdValue1513 = "curl might erroneously pass on credentials for a first proxy to a second\nproxy.\n\nThis can happen when the following conditions are true:\n\n1. curl is setup to use specific different proxies for different URL schemes\n2. the first proxy needs credentials\n3. the second proxy uses no credentials\n4. while using the first proxy (using say `http://`), curl is asked to follow\n   a redirect to a URL using another scheme (say `https://`), accessed using a\n   second, different, proxy"

var nvdValue1514 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1513}

var nvdValue1515 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1514}, ID: "CVE-2026-6253", LastModified: "2026-09-15T07:16:29.073", Metrics: &nvdValue1233, Published: "2026-05-13T13:01:56.570"}

var nvdValue1516 = "When asked to both use a `.netrc` file for credentials and to follow HTTP\nredirects, libcurl could leak the password used for the first host to the\nfollowed-to host under certain circumstances."

var nvdValue1517 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1516}

var nvdValue1518 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1517}, ID: "CVE-2026-6429", LastModified: "2026-09-15T07:16:29.557", Metrics: &nvdValue1233, Published: "2026-05-13T13:01:56.930"}

var nvdValue1519 = "Successfully using libcurl to do a transfer over a specific HTTP proxy\n(`proxyA`) with **Digest** authentication and then changing the proxy host to\na second one (`proxyB`) for a second transfer, reusing the same handle, makes\nlibcurl wrongly pass on the `Proxy-Authorization:` header field meant for\n`proxyA`, to `proxyB`."

var nvdValue1520 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1519}

var nvdValue1521 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue458}}

var nvdValue1522 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1520}, ID: "CVE-2026-7168", LastModified: "2026-09-15T07:16:29.890", Metrics: &nvdValue1521, Published: "2026-05-13T13:01:57.200"}

var nvdValue1523 = "Previously, CVE-2024-45337 fixed an authorization bypass for misused ssh server configurations; if any other type of callback is passed other than public key, then the source-address validation would be skipped."

var nvdValue1524 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1523}

var nvdValue1525 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1524}, ID: "CVE-2026-46595", LastModified: "2026-09-11T13:18:12.367", Metrics: &nvdValue1233, Published: "2026-05-22T04:16:25.550"}

var nvdValue1526 = "libexpat before 2.8.2 lacks handler call depth tracking for calls to XML_GetBuffer, XML_Parse, XML_ParseBuffer, XML_ParserFree, or XML_ParserReset from within handlers in cases of a policy violation. Thus, a use-after-free can occur,"

var nvdValue1527 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1526}

var nvdValue1528 = schema.CVSSV31{BaseScore: 5.9, VectorString: "CVSS:3.1/AV:L/AC:L/PR:N/UI:N/S:U/C:L/I:L/A:L", Version: "3.1"}

var nvdValue1529 = schema.CVEAPIJSON20CVSSV31{CvssData: &nvdValue1528}

var nvdValue1530 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue1529}}

var nvdValue1531 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1527}, ID: "CVE-2026-50219", LastModified: "2026-07-22T20:10:00.127", Metrics: &nvdValue1530, Published: "2026-06-04T06:16:25.050"}

var nvdValue1532 = "Issue summary: Parsing a crafted DER-encoded ASN.1 structure with a primitive\nelement whose content exceeds 2 gigabytes in length may cause a heap buffer\nover-read on 64-bit Unix and Unix-like platforms.\n\nImpact summary: The heap buffer over-read may crash the application (Denial of\nService) or to load into the decoded ASN.1 object contents of memory beyond the\nend of the input buffer.  More typically such ASN.1 elements would instead be\ntruncated.\n\nAn integer truncation in OpenSSL's ASN.1 decoder causes the content length of\nan ASN.1 primitive element to be mishandled when it exceeds 2 gigabytes. In the\nworst case the truncated length is treated as a request to scan the binary\ncontent for a terminating zero byte, possibly causing OpenSSL to read either\nless than or beyond the end of the allocated buffer.\n\nApplications that pass attacker-supplied data to d2i_X509(), d2i_PKCS7(), or\nany other d2i_* decoding function are affected. OpenSSL's own command-line\ntools are not vulnerable, as data read through the BIO layer is checked before\nit reaches the affected code. The issue only affects 64-bit Unix and Unix-like\nplatforms; 32-bit platforms and 64-bit Windows are not affected.\n\nThe FIPS modules in 4.0, 3.6, 3.5, 3.4 and 3.0 are not affected by this issue,\nas the affected code is outside the OpenSSL FIPS module boundary."

var nvdValue1533 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1532}

var nvdValue1534 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1533}, ID: "CVE-2026-34180", LastModified: "2026-07-23T08:10:00.137", Metrics: &nvdValue1233, Published: "2026-06-09T17:17:04.600"}

var nvdValue1535 = "Issue summary: A specially crafted password-encrypted CMS message\ncan trigger a NULL pointer dereference during CMS decryption.\n\nImpact summary: This NULL pointer dereference leads to an application crash\nand a Denial of Service.\n\nThe CMS PasswordRecipientInfo.keyDerivationAlgorithm field is defined as\nOPTIONAL in the ASN.1 specification and may therefore be absent in specially\ncrafted inputs. During the password-based CMS decryption the OpenSSL\nCMS implementation dereferences this field without first checking whether it\nwas present.\n\nAn attacker who supplies such a CMS message to an application performing\npassword-based CMS decryption can trigger an application crash, leading to\na Denial of Service.\n\nApplications that process password-encrypted CMS messages may be affected.\n\nThe FIPS modules in 4.0, 3.6, 3.5, 3.4, and 3.0 are not affected by this\nissue, as the affected code is outside the OpenSSL FIPS module boundary."

var nvdValue1536 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1535}

var nvdValue1537 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1536}, ID: "CVE-2026-42766", LastModified: "2026-07-23T08:10:00.137", Metrics: &nvdValue1233, Published: "2026-06-09T17:17:07.970"}

var nvdValue1538 = "Issue summary: A specially crafted PKCS#7 or S/MIME signed message could\ntrigger a use-after-free during PKCS#7 signature verification.\n\nImpact summary: A use-after-free may result in process crashes, heap\ncorruption, or potentially remote code execution.\n\nWhen processing a PKCS#7 or S/MIME signed message, if the SignedData\ndigestAlgorithms field is present as an empty ASN.1 SET, OpenSSL may\nincorrectly free a caller-owned BIO during PKCS7_verify(). A subsequent\nuse of the BIO by the calling application results in a use-after-free\ncondition.\n\nIn the common case this occurs when the application later calls\nBIO_free() on the BIO originally passed to PKCS7_verify(). Depending\non allocator behavior and application-specific BIO usage patterns, this\nmay result in a crash or other memory corruption. In some application\ncontexts this may potentially be exploitable for remote code execution.\n\nApplications that process PKCS#7 or S/MIME signed messages using OpenSSL\nPKCS#7 APIs may be affected. Applications using the CMS APIs for this\nprocessing are not affected.\n\nThe FIPS modules in 4.0, 3.6, 3.5, 3.4, and 3.0 are not affected by this\nissue, as the affected code is outside the OpenSSL FIPS module boundary."

var nvdValue1539 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1538}

var nvdValue1540 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1539}, ID: "CVE-2026-45447", LastModified: "2026-09-18T13:18:26.343", Metrics: &nvdValue1233, Published: "2026-06-09T17:17:19.277"}

var nvdValue1541 = "Issue summary: A signed integer overflow when sizing the destination\nbuffer for Unicode output in ASN1_mbstring_ncopy() can lead to a heap\nbuffer overflow.\n\nImpact summary: A heap buffer overflow may lead to a crash or possibly\nattacker controlled code execution or other undefined behaviour.\n\nIn ASN1_mbstring_copy() and ASN1_mbstring_ncopy() the destination\nsize for Unicode output is computed in a signed int: by left shift\nof the input character count for BMPSTRING (UTF-16) and\nUNIVERSALSTRING (UTF-32), and by summing per-character byte counts\nfor UTF8STRING. The calculation overflows when the input reaches\naround 2^30 characters. In the worst case (UNIVERSALSTRING at 2^30\ncharacters) the size wraps to zero, OPENSSL_malloc(1) is called, and\nthe subsequent character copy writes several gigabytes past the\none-byte allocation.\n\nX.509 certificate processing routes through ASN1_STRING_set_by_NID(),\nwhose DIRSTRING_TYPE mask excludes UNIVERSALSTRING and whose per-NID\nsize limits cap the input length; no network protocol or\ncertificate-handling path in OpenSSL exercises the overflow.\nTriggering the bug requires an application that calls\nASN1_mbstring_copy() or ASN1_mbstring_ncopy() directly, or registers\na custom string type via ASN1_STRING_TABLE_add(), with\nattacker-controlled input on the order of half a gigabyte or more.\nFor these reasons this issue was assigned Low severity.\n\nThe FIPS modules in 4.0, 3.6, 3.5, 3.4 and 3.0 are not affected by\nthis issue, as the affected code is outside the OpenSSL FIPS module\nboundary."

var nvdValue1542 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1541}

var nvdValue1543 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1542}, ID: "CVE-2026-7383", LastModified: "2026-07-23T08:10:00.137", Metrics: &nvdValue1233, Published: "2026-06-09T17:17:50.337"}

var nvdValue1544 = "Issue summary: When CMS password-based decryption (RFC 3211 / PWRI key unwrap)\nprocesses attacker-supplied CMS data, an attacker-chosen stream-mode KEK\ncipher can trigger a heap out-of-bounds read in kek_unwrap_key().\n\nImpact summary: A heap buffer over-read may trigger a crash which leads to\nDenial of Service for an application if the input buffer ends at a memory\npage boundary and the following page is unmapped. There is no information\ndisclosure as the over-read bytes are not revealed to the attacker.\n\nThe key unwrapping function performs a check-byte test as specified in the\nRFC that reads 7 bytes from a heap allocation that is based on the wrapped\nkey length from the message. There is a minimum length check based on the\nblock length of the wrapping cipher. However the cipher is selected from\nan OID carried in the attacker's PWRI keyEncryptionAlgorithm with no\nrequirement that the cipher be a block cipher. When an attacker selects\na stream-mode cipher the guard will be ineffective and the allocated buffer\ncontaining the unwrapped key can be too small to fit the check-bytes\nspecified in the RFC and a buffer over-read can happen.\n\nApplications calling CMS_decrypt() or CMS_decrypt_set1_password()\n(equivalently openssl cms -decrypt -pwri_password ...) on untrusted CMS\ndata are vulnerable to this issue. No password knowledge is required: the\nover-read happens during the unwrap attempt before any authentication\nsucceeds.\n\nThe over-read is limited to a few bytes and is not written to output, so\nthere is no information disclosure. Triggering a crash requires the\nallocation to border unmapped memory, which is unlikely with the normal\nallocator.\n\nThe FIPS modules are not affected by this issue."

var nvdValue1545 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1544}

var nvdValue1546 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1545}, ID: "CVE-2026-9076", LastModified: "2026-07-23T08:10:00.137", Metrics: &nvdValue1233, Published: "2026-06-09T17:17:50.997"}

var nvdValue1547 = "libexpat before 2.8.2 lacks handler call depth tracking for calls to XML_ResumeParser from within handlers in cases of a policy violation. Thus, a use-after-free can occur (similar to the CVE-2026-50219 situation)."

var nvdValue1548 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1547}

var nvdValue1549 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1548}, ID: "CVE-2026-56131", LastModified: "2026-06-23T20:15:48.007", Metrics: &nvdValue1233, Published: "2026-06-19T06:17:10.107"}

var nvdValue1550 = "In libexpat before 2.8.2, there is a heap-based buffer overflow in doProlog in xmlparse.c because scaffold backing array reallocation is mishandled when there is data-structure sharing across parsers."

var nvdValue1551 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1550}

var nvdValue1552 = schema.CVSSV31{BaseScore: 6.9, VectorString: "CVSS:3.1/AV:L/AC:H/PR:N/UI:N/S:U/C:H/I:H/A:L", Version: "3.1"}

var nvdValue1553 = schema.CVEAPIJSON20CVSSV31{CvssData: &nvdValue1552}

var nvdValue1554 = schema.CVEAPIJSON20CVEItemMetrics{CvssMetricV31: []*schema.CVEAPIJSON20CVSSV31{&nvdValue1553}}

var nvdValue1555 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1551}, ID: "CVE-2026-56132", LastModified: "2026-06-23T20:15:26.230", Metrics: &nvdValue1554, Published: "2026-06-19T06:17:10.253"}

var nvdValue1556 = schema.CVEAPIJSON20LangString{Lang: "en", Value: "libexpat before 2.8.2 has an integer overflow in storeAtts."}

var nvdValue1557 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1556}, ID: "CVE-2026-56403", LastModified: "2026-06-23T20:15:16.760", Metrics: &nvdValue1554, Published: "2026-06-21T16:16:26.590"}

var nvdValue1558 = schema.CVEAPIJSON20LangString{Lang: "en", Value: "libexpat before 2.8.2 has an integer overflow in addBinding."}

var nvdValue1559 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1558}, ID: "CVE-2026-56404", LastModified: "2026-06-23T20:15:05.850", Metrics: &nvdValue1554, Published: "2026-06-21T16:16:27.620"}

var nvdValue1560 = schema.CVEAPIJSON20LangString{Lang: "en", Value: "libexpat before 2.8.2 has an integer overflow in getAttributeId."}

var nvdValue1561 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1560}, ID: "CVE-2026-56405", LastModified: "2026-06-23T20:14:51.730", Metrics: &nvdValue1554, Published: "2026-06-21T16:16:27.740"}

var nvdValue1562 = "libexpat before 2.8.2 has an integer overflow in XML_ParseBuffer because it lacked a check that was present in XML_Parse."

var nvdValue1563 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1562}

var nvdValue1564 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1563}, ID: "CVE-2026-56406", LastModified: "2026-06-23T16:29:06.077", Metrics: &nvdValue1233, Published: "2026-06-21T16:16:27.870"}

var nvdValue1565 = schema.CVEAPIJSON20LangString{Lang: "en", Value: "libexpat before 2.8.2 has an integer overflow in doProlog that is related to storeEntityValue and entity textLen."}

var nvdValue1566 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1565}, ID: "CVE-2026-56407", LastModified: "2026-06-23T16:28:29.983", Metrics: &nvdValue1233, Published: "2026-06-21T16:16:27.987"}

var nvdValue1567 = schema.CVEAPIJSON20LangString{Lang: "en", Value: "libexpat before 2.8.2 has an integer overflow in copyString."}

var nvdValue1568 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1567}, ID: "CVE-2026-56408", LastModified: "2026-06-23T16:27:26.523", Metrics: &nvdValue1233, Published: "2026-06-21T16:16:28.110"}

var nvdValue1569 = schema.CVEAPIJSON20LangString{Lang: "en", Value: "xmlwf in libexpat before 2.8.2 has an integer overflow for the output filename when -d outputDir is used."}

var nvdValue1570 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1569}, ID: "CVE-2026-56409", LastModified: "2026-06-23T16:21:55.607", Metrics: &nvdValue1233, Published: "2026-06-21T16:16:28.230"}

var nvdValue1571 = schema.CVEAPIJSON20LangString{Lang: "en", Value: "xmlwf in libexpat before 2.8.2 has an integer overflow in resolveSystemId."}

var nvdValue1572 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1571}, ID: "CVE-2026-56410", LastModified: "2026-06-23T16:18:16.427", Metrics: &nvdValue1233, Published: "2026-06-21T16:16:28.360"}

var nvdValue1573 = schema.CVEAPIJSON20LangString{Lang: "en", Value: "xmlwf in libexpat before 2.8.2 has an integer overflow in endDoctypeDecl via NOTATION declarations."}

var nvdValue1574 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1573}, ID: "CVE-2026-56411", LastModified: "2026-06-23T16:16:36.417", Metrics: &nvdValue1233, Published: "2026-06-21T17:16:44.523"}

var nvdValue1575 = "libexpat before 2.8.2 does not consider XML_TOK_DATA_CHARS in doCdataSection and thus lacks handler call depth tracking for various calls from within handlers in cases of a policy violation. Thus, a use-after-free can occur. NOTE: this issue exists because of an incomplete fix for CVE-2026-50219."

var nvdValue1576 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1575}

var nvdValue1577 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1576}, ID: "CVE-2026-56412", LastModified: "2026-06-23T15:31:30.853", Metrics: &nvdValue1530, Published: "2026-06-21T17:16:44.657"}

var nvdValue1578 = "Successfully using libcurl to do a transfer to a specific HTTP origin\n(`hostA`) with **Digest** authentication and then changing the origin to a\ndifferent one (`hostB`) for a second transfer, reusing the same handle, makes\nlibcurl wrongly pass on the `Authorization:` header field meant for `hostA`,\nto `hostB`."

var nvdValue1579 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1578}

var nvdValue1580 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1579}, ID: "CVE-2026-11856", LastModified: "2026-09-15T07:16:25.870", Metrics: &nvdValue1233, Published: "2026-07-03T07:16:23.973"}

var nvdValue1581 = "A vulnerability exists where a new transfer that uses STARTTLS to upgrade the\nconnection might reuse an existing live connection even though the TLS\nconfiguration mismatches so it should not."

var nvdValue1582 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1581}

var nvdValue1583 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1582}, ID: "CVE-2026-8286", LastModified: "2026-09-15T07:16:31.617", Metrics: &nvdValue1233, Published: "2026-07-03T07:16:24.453"}

var nvdValue1584 = "libcurl might in some circumstances reuse the wrong connection when asked to\ndo Negotiate-authenticated ones, even when they are set to use different\n\"services\".\n\nlibcurl features a pool of recent connections so that subsequent requests can\nreuse an existing connection to avoid overhead.\n\nWhen reusing a connection a range of criteria must be met. Due to a logical\nerror in the code, a request that was issued by an application could\nwrongfully reuse an existing connection to the same server that was\nauthenticated using different services."

var nvdValue1585 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1584}

var nvdValue1586 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1585}, ID: "CVE-2026-8458", LastModified: "2026-09-15T07:16:32.327", Metrics: &nvdValue1233, Published: "2026-07-03T07:16:24.630"}

var nvdValue1587 = "A flaw in curl’s cookie parsing logic allows a malicious HTTP server to set\n\"super cookies\" that bypass the Public Suffix List check. This enables an\nattacker-controlled origin to inject cookies that curl subsequently scopes and\ntransmits to unrelated third-party domains."

var nvdValue1588 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1587}

var nvdValue1589 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1588}, ID: "CVE-2026-8924", LastModified: "2026-09-15T07:16:32.573", Metrics: &nvdValue1233, Published: "2026-07-03T07:16:24.793"}

var nvdValue1590 = "When reusing a libcurl handle for sequential transfers driven by\nenvironment-variable proxy configuration, libcurl fails to clear the proxy\nauthentication state between requests. Specifically, if the initial transfer\nauthenticates against `proxyA` using Digest auth, a subsequent transfer routed\nthrough `proxyB` erroneously leaks the `Proxy-Authorization:` header intended\nsolely for `proxyA`."

var nvdValue1591 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1590}

var nvdValue1592 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1591}, ID: "CVE-2026-8927", LastModified: "2026-09-15T07:16:33.157", Metrics: &nvdValue1233, Published: "2026-07-03T07:16:25.123"}

var nvdValue1593 = "libcurl would reuse a previously created connection even when some mTLS config\nrelated option had been changed that should have prohibited reuse.\n\nlibcurl keeps previously used connections in a connection pool for subsequent\ntransfers to reuse if one of them matches the setup. However, some TLS\nsettings related to client certificates were left out from the configuration\nmatch checks, making them match too easily. In particular options related to\nthe private key."

var nvdValue1594 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1593}

var nvdValue1595 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1594}, ID: "CVE-2026-8932", LastModified: "2026-09-15T07:16:33.407", Metrics: &nvdValue1233, Published: "2026-07-03T07:16:25.363"}

var nvdValue1596 = "libexpat before 2.8.3 has an out-of-bounds read and resultant infinite loop because low surrogates are treated the same as high surrogates during Unicode processing in the *_toUtf16 functions."

var nvdValue1597 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1596}

var nvdValue1598 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1597}, ID: "CVE-2026-72522", LastModified: "2026-08-31T19:33:11.197", Metrics: &nvdValue1233, Published: "2026-08-10T04:16:50.910"}

var nvdValue1599 = "Expat through 2.8.3 contains a denial of service vulnerability caused by quadratic algorithmic complexity in the storeAtts() function in xmlparse.c, where processing N specified attributes with non-normalized values triggers an O(N^2) linear scan of elementType->defaultAtts to determine CDATA status. A remote unauthenticated attacker can supply a single well-formed XML document of a few megabytes to an application parsing untrusted XML to cause excessive CPU consumption, resulting in denial of service without requiring authentication, external entity resolution, or non-default parser options."

var nvdValue1600 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1599}

var nvdValue1601 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1600}, ID: "CVE-2026-66046", LastModified: "2026-09-18T15:07:24.370", Metrics: &nvdValue1233, Published: "2026-08-18T15:16:57.000"}

var nvdValue1602 = "libexpat before 2.8.4 lacks handler call depth tracking with custom encoding callbacks. Thus, a use-after-free can occur. NOTE: this is similar to CVE-2026-50219, CVE-2026-56131 and CVE-2026-56412."

var nvdValue1603 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1602}

var nvdValue1604 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1603}, ID: "CVE-2026-76957", LastModified: "2026-09-08T20:56:31.860", Metrics: &nvdValue1173, Published: "2026-08-20T05:16:29.747"}

var nvdValue1605 = "Expat through 2.8.3 contains an out-of-bounds read vulnerability that allows attackers to trigger memory corruption by processing XML with external entity parsers created via XML_ExternalEntityParserCreate. A struct size mismatch between ELEMENT_TYPE members causes storeAtts to read the attIndex member past allocated memory boundaries, resulting in failure to normalize whitespace in non-CDATA attributes or a wild pointer dereference causing a segfault. This vulnerability was introduced by the fix for CVE-2026-66046."

var nvdValue1606 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1605}

var nvdValue1607 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1606}, ID: "CVE-2026-76641", LastModified: "2026-08-20T19:17:04.430", Metrics: &nvdValue1233, Published: "2026-08-20T18:16:51.887"}

var nvdValue1608 = "Issue summary: Receiving a DTLS record for a future epoch while a handshake\nis in progress causes OpenSSL to buffer far more memory than the record\nitself requires.\n\nImpact summary: A peer can use a small amount of network traffic to make an\nOpenSSL DTLS endpoint retain a disproportionately large amount of memory,\nwhich may lead to a Denial of Service.\n\nCWE: CWE-405: Asymmetric Resource Consumption (Amplification)\n\nDescription: While a DTLS handshake is in progress, a peer may legitimately\nhave already moved on to the next epoch (for example, having sent its\nChangeCipherSpec and Finished messages) before the local endpoint has\nprocessed the same transition, typically because of reordering on the\nunderlying UDP transport. OpenSSL buffers such early records so that they\ncan be processed once the local endpoint catches up.\n\nBuffering a record currently retains the entire read buffer it arrived in,\nwhich is sized to hold the largest possible DTLS record (around 16\nkilobytes), rather than just the bytes that make up the record itself. Up\nto 100 such records may be buffered per connection. As a result, a peer\nthat sends a stream of small forged records claiming to belong to the next\nepoch can cause an OpenSSL DTLS endpoint to retain around 1.7 megabytes of\nmemory, despite sending only a small fraction of that amount of data over\nthe network.\n\nAn attacker therefore gains a memory amplification factor of around 1200,\nand can multiply the effect across as many associations as it is able to\nopen, making this a remote memory exhaustion Denial of Service risk for\nDTLS servers. Since the memory retained per connection remains bounded,\nand any limit an application already places on the number of concurrent\nassociations also bounds the total exposure, this issue has been assessed\nas Low severity.\n\nFIPS impact: no\n\nNo FIPS modules are affected by this issue as the affected code is outside\nthe OpenSSL FIPS module boundary.\n\nOpenSSL 4.0, 3.6, 3.5, 3.4, 3.0, 1.1.1 and 1.0.2 are vulnerable to this\nissue.\n\nOpenSSL 4.0 users should upgrade to OpenSSL 4.0.2.\nOpenSSL 3.6 users should upgrade to OpenSSL 3.6.4.\nOpenSSL 3.5 users should upgrade to OpenSSL 3.5.8.\nOpenSSL 3.4 users should upgrade to OpenSSL 3.4.7.\nOpenSSL 3.0 users should upgrade to OpenSSL 3.0.22.\n\nPremium support customers only:\nOpenSSL 1.1.1 users should upgrade to OpenSSL 1.1.1zi\nOpenSSL 1.0.2 users should upgrade to OpenSSL 1.0.2zr\n\nThis issue was reported on 18 May 2026 by Amazon Web Services.\nThe fix has been developed by Matt Caswell.\n\n-- cut (non-publishing metadata for internal use) --\nReported by: Amazon Web Services\nFixed by: Matt Caswell"

var nvdValue1609 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1608}

var nvdValue1610 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1609}, ID: "CVE-2026-54874", LastModified: "2026-09-11T21:16:28.067", Metrics: &nvdValue1233, Published: "2026-08-25T13:19:24.033"}

var nvdValue1611 = "Issue summary: OpenSSL CMS decryption sizes the key-unwrap output buffer based\non querying the unwrapped key size, but the AES-WRAP-PAD unwrap primitive\ncan write and cleanse more bytes than that query reports, causing an 8-byte\nout-of-bounds heap write.\n\nImpact summary: An attacker who supplies a crafted CMS message can trigger a\ndeterministic 8-byte out-of-bounds heap write when the victim decrypts it\nwith CMS_decrypt(), corrupting the heap and typically resulting in a Denial\nof Service.\n\nCWE: CWE-787: Out-of-bounds Write\n\nDescription: The key-wrap OID is potentially attacker-controlled on the wire.\nCMS unwrapping allows both id-aesNNN-wrap-pad and id-aesNNN-wrap ciphers.\nAn attacker can take a legitimate message and change a single OID byte to\nselect the padded variant while leaving the message otherwise valid. Since\nthe unwrap key is derived from the recipient's private operation (ECDH key\nagreement or ML-KEM decapsulation), the RFC 5649 integrity check cannot\npass, and the decryption fails with integrity failure.\n\nThe write is a fixed-size (8-byte), fixed-value (zero) heap overflow\nimmediately past the allocation, requires no special configuration, and is\nreachable from the public CMS_decrypt() function. The consequence is\na heap corruption leading to a Denial of Service. The fix in the CMS code\nsizes the unwrap output buffer for the worst case so a failed unwrap cannot\nwrite past the allocation.\n\nFIPS impact: no\n\nAs the CMS code lives outside the FIPS module boundary, no FIPS\nmodules are affected by this CVE."

var nvdValue1612 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1611}

var nvdValue1613 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1612}, ID: "CVE-2026-63072", LastModified: "2026-09-11T21:16:34.287", Metrics: &nvdValue1233, Published: "2026-08-25T13:19:26.010"}

var nvdValue1614 = "A flaw in libcurl's handling of HTTP/2 Server Push streams, when the parent\nhandle is set to share connections with other handles, can lead to\nuse-after-free in the cleanup process."

var nvdValue1615 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1614}

var nvdValue1616 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1615}, ID: "CVE-2026-18924", LastModified: "2026-09-15T07:16:27.063", Metrics: &nvdValue1233, Published: "2026-09-06T18:17:20.553"}

var nvdValue1617 = "A flaw in libcurl makes it wrongly reuse an HTTP connection setup for a given\nhostname using Negotiate authentication, when the initial request is done\nusing empty credentials. This can make user B's request get sent over user A's\npreviously authenticated connection."

var nvdValue1618 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1617}

var nvdValue1619 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1618}, ID: "CVE-2026-19931", LastModified: "2026-09-15T07:16:27.290", Metrics: &nvdValue1233, Published: "2026-09-06T18:17:20.733"}

var nvdValue1620 = "When `CURLOPT_PINNEDPUBLICKEY` is configured alongside options that disable\nstandard peer verification (`CURLOPT_SSL_VERIFYPEER = 0` and\n`CURLOPT_SSL_VERIFYHOST = 0`), libcurl fails to enforce public key pinning on\nconnections established without a presented server certificate. Bypassing the\npinning check under these disabled-verification conditions allows\nunauthenticated connections to succeed when they should be rejected."

var nvdValue1621 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1620}

var nvdValue1622 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1621}, ID: "CVE-2026-80230", LastModified: "2026-09-15T07:16:30.337", Metrics: &nvdValue1233, Published: "2026-09-06T18:17:22.327"}

var nvdValue1623 = "When libpsl support is enabled, libcurl fails to enforce the Public Suffix\nList boundary check when processing a `Set-Cookie` header where the `Domain`\nattribute explicitly matches an origin host that is itself a public suffix\n(e.g., `Domain=co.uk` set by `co.uk`).\n\nInstead of coercing it into a strict host-only cookie, libcurl saves the\ncookie with wildcard domain scope (`.co.uk`). Consequently, the cookie is\ninappropriately included in subsequent outbound requests or HTTP redirects to\narbitrary sibling subdomains under the same public suffix (e.g.,\n`attacker.co.uk`)."

var nvdValue1624 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1623}

var nvdValue1625 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1624}, ID: "CVE-2026-82209", LastModified: "2026-09-15T07:16:31.233", Metrics: &nvdValue1233, Published: "2026-09-06T18:17:22.847"}

var nvdValue1626 = "Expat through 2.8.4 fails to validate low surrogates following high surrogates in UTF-16 input, allowing malformed UTF-16 sequences to be accepted. Attackers can craft UTF-16 encoded XML with lone high surrogates that consume following code units, hiding markup characters from the parser and enabling XML injection attacks."

var nvdValue1627 = schema.CVEAPIJSON20LangString{Lang: "en", Value: nvdValue1626}

var nvdValue1628 = schema.CVEAPIJSON20CVEItem{Descriptions: []*schema.CVEAPIJSON20LangString{&nvdValue1627}, ID: "CVE-2026-93990", LastModified: "2026-09-19T23:17:10.203", Metrics: &nvdValue1132, Published: "2026-09-19T23:17:10.203"}
