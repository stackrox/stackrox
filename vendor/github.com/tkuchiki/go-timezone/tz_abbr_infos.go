package timezone

var tzAbbrInfos = map[string][]*TzAbbreviationInfo{
	"GMT": []*TzAbbreviationInfo{
		{
			countryCode: "",
			isDST:       false,
			name:        "Greenwich Mean Time",
			offset:      0,
			offsetHHMM:  "+00:00",
		},
	},
	"GHST": []*TzAbbreviationInfo{
		{
			countryCode: "GH",
			isDST:       true,
			name:        "Ghana Summer Time",
			offset:      1200,
			offsetHHMM:  "+00:20",
		},
	},
	"EAT": []*TzAbbreviationInfo{
		{
			countryCode: "ET",
			isDST:       false,
			name:        "East Africa Time",
			offset:      10800,
			offsetHHMM:  "+03:00",
		},
	},
	"CET": []*TzAbbreviationInfo{
		{
			countryCode: "DZ",
			isDST:       false,
			name:        "Central European Time/Central European Standard Time",
			offset:      3600,
			offsetHHMM:  "+01:00",
		},
	},
	"WAT": []*TzAbbreviationInfo{
		{
			countryCode: "CF",
			isDST:       false,
			name:        "West Africa Time/West Africa Standard Time",
			offset:      3600,
			offsetHHMM:  "+01:00",
		},
	},
	"CAT": []*TzAbbreviationInfo{
		{
			countryCode: "MW",
			isDST:       false,
			name:        "Central Africa Time",
			offset:      7200,
			offsetHHMM:  "+02:00",
		},
	},
	"EET": []*TzAbbreviationInfo{
		{
			countryCode: "EG",
			isDST:       false,
			name:        "Eastern European Time/Eastern European Standard Time",
			offset:      7200,
			offsetHHMM:  "+02:00",
		},
	},
	"EEST": []*TzAbbreviationInfo{
		{
			countryCode: "EG",
			isDST:       true,
			name:        "Eastern European Summer Time",
			offset:      10800,
			offsetHHMM:  "+03:00",
		},
	},
	"WET": []*TzAbbreviationInfo{
		{
			countryCode: "MA",
			isDST:       false,
			name:        "Western European Time/Western European Standard Time",
			offset:      0,
			offsetHHMM:  "+00:00",
		},
		{
			countryCode: "",
			isDST:       false,
			name:        "Western European Standard Time",
			offset:      0,
			offsetHHMM:  "+00:00",
		},
	},
	"WEST": []*TzAbbreviationInfo{
		{
			countryCode: "MA",
			isDST:       true,
			name:        "Western European Summer Time",
			offset:      3600,
			offsetHHMM:  "+01:00",
		},
	},
	"CEST": []*TzAbbreviationInfo{
		{
			countryCode: "ES",
			isDST:       true,
			name:        "Central European Summer Time",
			offset:      7200,
			offsetHHMM:  "+02:00",
		},
	},
	"SAST": []*TzAbbreviationInfo{
		{
			countryCode: "ZA",
			isDST:       false,
			name:        "South Africa Standard Time",
			offset:      7200,
			offsetHHMM:  "+02:00",
		},
		{
			countryCode: "ZA",
			isDST:       true,
			name:        "South Africa Summer Time",
			offset:      10800,
			offsetHHMM:  "+03:00",
		},
	},
	"CAST": []*TzAbbreviationInfo{
		{
			countryCode: "SD",
			isDST:       true,
			name:        "Central Africa Summer Time",
			offset:      10800,
			offsetHHMM:  "+03:00",
		},
	},
	"HAT": []*TzAbbreviationInfo{
		{
			countryCode: "US",
			isDST:       false,
			name:        "Hawaii-Aleutian Time",
			offset:      -36000,
			offsetHHMM:  "-10:00",
		},
	},
	"HAST": []*TzAbbreviationInfo{
		{
			countryCode: "US",
			isDST:       false,
			name:        "Hawaii-Aleutian Standard Time",
			offset:      -36000,
			offsetHHMM:  "-10:00",
		},
	},
	"HADT": []*TzAbbreviationInfo{
		{
			countryCode: "US",
			isDST:       true,
			name:        "Hawaii-Aleutian Daylight Time",
			offset:      -32400,
			offsetHHMM:  "-09:00",
		},
	},
	"AKT": []*TzAbbreviationInfo{
		{
			countryCode: "US",
			isDST:       false,
			name:        "Alaska Time",
			offset:      -32400,
			offsetHHMM:  "-09:00",
		},
	},
	"AKST": []*TzAbbreviationInfo{
		{
			countryCode: "US",
			isDST:       false,
			name:        "Alaska Standard Time",
			offset:      -32400,
			offsetHHMM:  "-09:00",
		},
	},
	"AKDT": []*TzAbbreviationInfo{
		{
			countryCode: "US",
			isDST:       true,
			name:        "Alaska Daylight Time",
			offset:      -28800,
			offsetHHMM:  "-08:00",
		},
	},
	"AT": []*TzAbbreviationInfo{
		{
			countryCode: "AI",
			isDST:       false,
			name:        "Atlantic Time",
			offset:      -14400,
			offsetHHMM:  "-04:00",
		},
	},
	"AST": []*TzAbbreviationInfo{
		{
			countryCode: "AI",
			isDST:       false,
			name:        "Atlantic Standard Time",
			offset:      -14400,
			offsetHHMM:  "-04:00",
		},
		{
			countryCode: "YE",
			isDST:       false,
			name:        "Arabian Time/Arabian Standard Time",
			offset:      10800,
			offsetHHMM:  "+03:00",
		},
	},
	"BRT": []*TzAbbreviationInfo{
		{
			countryCode: "BR",
			isDST:       false,
			name:        "Brasilia Time/Brasilia Standard Time",
			offset:      -10800,
			offsetHHMM:  "-03:00",
		},
	},
	"BRST": []*TzAbbreviationInfo{
		{
			countryCode: "BR",
			isDST:       true,
			name:        "Brasilia Summer Time",
			offset:      -7200,
			offsetHHMM:  "-02:00",
		},
	},
	"ART": []*TzAbbreviationInfo{
		{
			countryCode: "AR",
			isDST:       false,
			name:        "Argentina Time/Argentina Standard Time",
			offset:      -10800,
			offsetHHMM:  "-03:00",
		},
	},
	"ARST": []*TzAbbreviationInfo{
		{
			countryCode: "AR",
			isDST:       true,
			name:        "Argentina Summer Time",
			offset:      -7200,
			offsetHHMM:  "-02:00",
		},
	},
	"PYT": []*TzAbbreviationInfo{
		{
			countryCode: "PY",
			isDST:       false,
			name:        "Paraguay Time/Paraguay Standard Time",
			offset:      -14400,
			offsetHHMM:  "-04:00",
		},
	},
	"PYST": []*TzAbbreviationInfo{
		{
			countryCode: "PY",
			isDST:       true,
			name:        "Paraguay Summer Time",
			offset:      -10800,
			offsetHHMM:  "-03:00",
		},
	},
	"ET": []*TzAbbreviationInfo{
		{
			countryCode: "CA",
			isDST:       false,
			name:        "Eastern Time",
			offset:      -18000,
			offsetHHMM:  "-05:00",
		},
	},
	"EST": []*TzAbbreviationInfo{
		{
			countryCode: "CA",
			isDST:       false,
			name:        "Eastern Standard Time",
			offset:      -18000,
			offsetHHMM:  "-05:00",
		},
	},
	"CT": []*TzAbbreviationInfo{
		{
			countryCode: "MX",
			isDST:       false,
			name:        "Central Time",
			offset:      -21600,
			offsetHHMM:  "-06:00",
		},
		{
			countryCode: "CU",
			isDST:       false,
			name:        "Cuba Time",
			offset:      -18000,
			offsetHHMM:  "-05:00",
		},
		{
			countryCode: "CN",
			isDST:       false,
			name:        "China Time",
			offset:      28800,
			offsetHHMM:  "+08:00",
		},
		{
			countryCode: "TW",
			isDST:       false,
			name:        "Taipei Time",
			offset:      28800,
			offsetHHMM:  "+08:00",
		},
	},
	"CST": []*TzAbbreviationInfo{
		{
			countryCode: "MX",
			isDST:       false,
			name:        "Central Standard Time",
			offset:      -21600,
			offsetHHMM:  "-06:00",
		},
		{
			countryCode: "CU",
			isDST:       false,
			name:        "Cuba Standard Time",
			offset:      -18000,
			offsetHHMM:  "-05:00",
		},
		{
			countryCode: "CN",
			isDST:       false,
			name:        "China Standard Time",
			offset:      28800,
			offsetHHMM:  "+08:00",
		},
		{
			countryCode: "TW",
			isDST:       false,
			name:        "Taipei Standard Time",
			offset:      28800,
			offsetHHMM:  "+08:00",
		},
	},
	"CDT": []*TzAbbreviationInfo{
		{
			countryCode: "MX",
			isDST:       true,
			name:        "Central Daylight Time",
			offset:      -18000,
			offsetHHMM:  "-05:00",
		},
		{
			countryCode: "CU",
			isDST:       true,
			name:        "Cuba Daylight Time",
			offset:      -14400,
			offsetHHMM:  "-04:00",
		},
		{
			countryCode: "CN",
			isDST:       true,
			name:        "China Daylight Time",
			offset:      32400,
			offsetHHMM:  "+09:00",
		},
		{
			countryCode: "TW",
			isDST:       true,
			name:        "Taipei Daylight Time",
			offset:      32400,
			offsetHHMM:  "+09:00",
		},
	},
	"ADT": []*TzAbbreviationInfo{
		{
			countryCode: "BB",
			isDST:       true,
			name:        "Atlantic Daylight Time",
			offset:      -10800,
			offsetHHMM:  "-03:00",
		},
		{
			countryCode: "IQ",
			isDST:       true,
			name:        "Arabian Daylight Time",
			offset:      14400,
			offsetHHMM:  "+04:00",
		},
	},
	"AMT": []*TzAbbreviationInfo{
		{
			countryCode: "BR",
			isDST:       false,
			name:        "Amazon Time/Amazon Standard Time",
			offset:      -14400,
			offsetHHMM:  "-04:00",
		},
		{
			countryCode: "AM",
			isDST:       false,
			name:        "Armenia Time/Armenia Standard Time",
			offset:      14400,
			offsetHHMM:  "+04:00",
		},
	},
	"AMST": []*TzAbbreviationInfo{
		{
			countryCode: "BR",
			isDST:       true,
			name:        "Amazon Summer Time",
			offset:      -10800,
			offsetHHMM:  "-03:00",
		},
		{
			countryCode: "AM",
			isDST:       true,
			name:        "Armenia Summer Time",
			offset:      18000,
			offsetHHMM:  "+05:00",
		},
	},
	"COT": []*TzAbbreviationInfo{
		{
			countryCode: "CO",
			isDST:       false,
			name:        "Colombia Time/Colombia Standard Time",
			offset:      -18000,
			offsetHHMM:  "-05:00",
		},
	},
	"COST": []*TzAbbreviationInfo{
		{
			countryCode: "CO",
			isDST:       true,
			name:        "Colombia Summer Time",
			offset:      -14400,
			offsetHHMM:  "-04:00",
		},
	},
	"MT": []*TzAbbreviationInfo{
		{
			countryCode: "US",
			isDST:       false,
			name:        "Mountain Time",
			offset:      -25200,
			offsetHHMM:  "-07:00",
		},
		{
			countryCode: "MX",
			isDST:       false,
			name:        "Mexican Pacific Time",
			offset:      -25200,
			offsetHHMM:  "-07:00",
		},
	},
	"MST": []*TzAbbreviationInfo{
		{
			countryCode: "US",
			isDST:       false,
			name:        "Mountain Standard Time",
			offset:      -25200,
			offsetHHMM:  "-07:00",
		},
		{
			countryCode: "MX",
			isDST:       false,
			name:        "Mexican Pacific Standard Time",
			offset:      -25200,
			offsetHHMM:  "-07:00",
		},
	},
	"MDT": []*TzAbbreviationInfo{
		{
			countryCode: "US",
			isDST:       true,
			name:        "Mountain Daylight Time",
			offset:      -21600,
			offsetHHMM:  "-06:00",
		},
		{
			countryCode: "MX",
			isDST:       true,
			name:        "Mexican Pacific Daylight Time",
			offset:      -21600,
			offsetHHMM:  "-06:00",
		},
	},
	"VET": []*TzAbbreviationInfo{
		{
			countryCode: "VE",
			isDST:       false,
			name:        "Venezuela Time",
			offset:      -14400,
			offsetHHMM:  "-04:00",
		},
	},
	"GFT": []*TzAbbreviationInfo{
		{
			countryCode: "GF",
			isDST:       false,
			name:        "French Guiana Time",
			offset:      -10800,
			offsetHHMM:  "-03:00",
		},
	},
	"PT": []*TzAbbreviationInfo{
		{
			countryCode: "CA",
			isDST:       false,
			name:        "Pacific Time",
			offset:      -25200,
			offsetHHMM:  "-07:00",
		},
	},
	"PST": []*TzAbbreviationInfo{
		{
			countryCode: "CA",
			isDST:       false,
			name:        "Pacific Standard Time",
			offset:      -25200,
			offsetHHMM:  "-07:00",
		},
		{
			countryCode: "PN",
			isDST:       false,
			name:        "Pitcairn Time",
			offset:      -28800,
			offsetHHMM:  "-08:00",
		},
	},
	"EDT": []*TzAbbreviationInfo{
		{
			countryCode: "US",
			isDST:       true,
			name:        "Eastern Daylight Time",
			offset:      -14400,
			offsetHHMM:  "-04:00",
		},
	},
	"ACT": []*TzAbbreviationInfo{
		{
			countryCode: "BR",
			isDST:       false,
			name:        "Acre Time/Acre Standard Time",
			offset:      -18000,
			offsetHHMM:  "-05:00",
		},
		{
			countryCode: "AU",
			isDST:       false,
			name:        "Central Australia Time",
			offset:      34200,
			offsetHHMM:  "+09:30",
		},
	},
	"ACST": []*TzAbbreviationInfo{
		{
			countryCode: "BR",
			isDST:       true,
			name:        "Acre Summer Time",
			offset:      -14400,
			offsetHHMM:  "-04:00",
		},
		{
			countryCode: "AU",
			isDST:       false,
			name:        "Australian Central Standard Time",
			offset:      34200,
			offsetHHMM:  "+09:30",
		},
	},
	"PDT": []*TzAbbreviationInfo{
		{
			countryCode: "MX",
			isDST:       true,
			name:        "Pacific Daylight Time",
			offset:      -25200,
			offsetHHMM:  "-07:00",
		},
	},
	"WGT": []*TzAbbreviationInfo{
		{
			countryCode: "GL",
			isDST:       false,
			name:        "West Greenland Time/West Greenland Standard Time",
			offset:      -10800,
			offsetHHMM:  "-03:00",
		},
	},
	"WGST": []*TzAbbreviationInfo{
		{
			countryCode: "GL",
			isDST:       true,
			name:        "West Greenland Summer Time",
			offset:      -7200,
			offsetHHMM:  "-02:00",
		},
	},
	"ECT": []*TzAbbreviationInfo{
		{
			countryCode: "EC",
			isDST:       false,
			name:        "Ecuador Time",
			offset:      -18000,
			offsetHHMM:  "-05:00",
		},
	},
	"GYT": []*TzAbbreviationInfo{
		{
			countryCode: "GY",
			isDST:       false,
			name:        "Guyana Time",
			offset:      -14400,
			offsetHHMM:  "-04:00",
		},
	},
	"BOT": []*TzAbbreviationInfo{
		{
			countryCode: "BO",
			isDST:       false,
			name:        "Bolivia Time",
			offset:      -14400,
			offsetHHMM:  "-04:00",
		},
	},
	"BST": []*TzAbbreviationInfo{
		{
			countryCode: "BO",
			isDST:       true,
			name:        "Bolivia Summer Time",
			offset:      -12756,
			offsetHHMM:  "-03:27",
		},
		{
			countryCode: "GB",
			isDST:       true,
			name:        "British Summer Time",
			offset:      3600,
			offsetHHMM:  "+01:00",
		},
		{
			countryCode: "PG",
			isDST:       false,
			name:        "Bougainville Standard Time",
			offset:      39600,
			offsetHHMM:  "+11:00",
		},
	},
	"PET": []*TzAbbreviationInfo{
		{
			countryCode: "PE",
			isDST:       false,
			name:        "Peru Time/Peru Standard Time",
			offset:      -18000,
			offsetHHMM:  "-05:00",
		},
	},
	"PEST": []*TzAbbreviationInfo{
		{
			countryCode: "PE",
			isDST:       true,
			name:        "Peru Summer Time",
			offset:      -14400,
			offsetHHMM:  "-04:00",
		},
	},
	"PMST": []*TzAbbreviationInfo{
		{
			countryCode: "PM",
			isDST:       false,
			name:        "St. Pierre & Miquelon Time/St. Pierre & Miquelon Standard Time",
			offset:      -10800,
			offsetHHMM:  "-03:00",
		},
	},
	"PMDT": []*TzAbbreviationInfo{
		{
			countryCode: "PM",
			isDST:       true,
			name:        "St. Pierre & Miquelon Daylight Time",
			offset:      -7200,
			offsetHHMM:  "-02:00",
		},
	},
	"UYT": []*TzAbbreviationInfo{
		{
			countryCode: "UY",
			isDST:       false,
			name:        "Uruguay Time/Uruguay Standard Time",
			offset:      -10800,
			offsetHHMM:  "-03:00",
		},
	},
	"UYST": []*TzAbbreviationInfo{
		{
			countryCode: "UY",
			isDST:       true,
			name:        "Uruguay Summer Time",
			offset:      -7200,
			offsetHHMM:  "-02:00",
		},
	},
	"FNT": []*TzAbbreviationInfo{
		{
			countryCode: "BR",
			isDST:       false,
			name:        "Fernando de Noronha Time/Fernando de Noronha Standard Time",
			offset:      -7200,
			offsetHHMM:  "-02:00",
		},
	},
	"FNST": []*TzAbbreviationInfo{
		{
			countryCode: "BR",
			isDST:       true,
			name:        "Fernando de Noronha Summer Time",
			offset:      -3600,
			offsetHHMM:  "-01:00",
		},
	},
	"SRT": []*TzAbbreviationInfo{
		{
			countryCode: "SR",
			isDST:       false,
			name:        "Suriname Time",
			offset:      -10800,
			offsetHHMM:  "-03:00",
		},
	},
	"CLT": []*TzAbbreviationInfo{
		{
			countryCode: "CL",
			isDST:       false,
			name:        "Chile Time/Chile Standard Time",
			offset:      -10800,
			offsetHHMM:  "-03:00",
		},
	},
	"CLST": []*TzAbbreviationInfo{
		{
			countryCode: "CL",
			isDST:       true,
			name:        "Chile Summer Time",
			offset:      -10800,
			offsetHHMM:  "-03:00",
		},
	},
	"EHDT": []*TzAbbreviationInfo{
		{
			countryCode: "DO",
			isDST:       true,
			name:        "Eastern Half Daylight Time",
			offset:      -16200,
			offsetHHMM:  "-04:30",
		},
	},
	"EGT": []*TzAbbreviationInfo{
		{
			countryCode: "GL",
			isDST:       false,
			name:        "East Greenland Time/East Greenland Standard Time",
			offset:      -3600,
			offsetHHMM:  "-01:00",
		},
	},
	"EGST": []*TzAbbreviationInfo{
		{
			countryCode: "GL",
			isDST:       true,
			name:        "East Greenland Summer Time",
			offset:      0,
			offsetHHMM:  "+00:00",
		},
	},
	"NT": []*TzAbbreviationInfo{
		{
			countryCode: "CA",
			isDST:       false,
			name:        "Newfoundland Time",
			offset:      -12600,
			offsetHHMM:  "-03:30",
		},
	},
	"NST": []*TzAbbreviationInfo{
		{
			countryCode: "CA",
			isDST:       false,
			name:        "Newfoundland Standard Time",
			offset:      -12600,
			offsetHHMM:  "-03:30",
		},
	},
	"NDT": []*TzAbbreviationInfo{
		{
			countryCode: "CA",
			isDST:       true,
			name:        "Newfoundland Daylight Time",
			offset:      -9000,
			offsetHHMM:  "-02:30",
		},
	},
	"AWT": []*TzAbbreviationInfo{
		{
			countryCode: "AQ",
			isDST:       false,
			name:        "Australian Western Time",
			offset:      28800,
			offsetHHMM:  "+08:00",
		},
	},
	"AWST": []*TzAbbreviationInfo{
		{
			countryCode: "AQ",
			isDST:       false,
			name:        "Australian Western Standard Time",
			offset:      28800,
			offsetHHMM:  "+08:00",
		},
	},
	"DAVT": []*TzAbbreviationInfo{
		{
			countryCode: "AQ",
			isDST:       false,
			name:        "Davis Time",
			offset:      25200,
			offsetHHMM:  "+07:00",
		},
	},
	"DDUT": []*TzAbbreviationInfo{
		{
			countryCode: "AQ",
			isDST:       false,
			name:        "Dumont-d’Urville Time",
			offset:      36000,
			offsetHHMM:  "+10:00",
		},
	},
	"MIST": []*TzAbbreviationInfo{
		{
			countryCode: "AU",
			isDST:       false,
			name:        "Macquarie Island Time",
			offset:      39600,
			offsetHHMM:  "+11:00",
		},
	},
	"MAWT": []*TzAbbreviationInfo{
		{
			countryCode: "AQ",
			isDST:       false,
			name:        "Mawson Time",
			offset:      18000,
			offsetHHMM:  "+05:00",
		},
	},
	"NZT": []*TzAbbreviationInfo{
		{
			countryCode: "AQ",
			isDST:       false,
			name:        "New Zealand Time",
			offset:      43200,
			offsetHHMM:  "+12:00",
		},
	},
	"NZST": []*TzAbbreviationInfo{
		{
			countryCode: "AQ",
			isDST:       false,
			name:        "New Zealand Standard Time",
			offset:      43200,
			offsetHHMM:  "+12:00",
		},
	},
	"NZDT": []*TzAbbreviationInfo{
		{
			countryCode: "AQ",
			isDST:       true,
			name:        "New Zealand Daylight Time",
			offset:      46800,
			offsetHHMM:  "+13:00",
		},
	},
	"ROTT": []*TzAbbreviationInfo{
		{
			countryCode: "AQ",
			isDST:       false,
			name:        "Rothera Time",
			offset:      -10800,
			offsetHHMM:  "-03:00",
		},
	},
	"SYOT": []*TzAbbreviationInfo{
		{
			countryCode: "AQ",
			isDST:       false,
			name:        "Syowa Time",
			offset:      10800,
			offsetHHMM:  "+03:00",
		},
	},
	"VOST": []*TzAbbreviationInfo{
		{
			countryCode: "AQ",
			isDST:       false,
			name:        "Vostok Time",
			offset:      21600,
			offsetHHMM:  "+06:00",
		},
	},
	"ALMT": []*TzAbbreviationInfo{
		{
			countryCode: "KZ",
			isDST:       false,
			name:        "Almaty Time/Almaty Standard Time",
			offset:      21600,
			offsetHHMM:  "+06:00",
		},
		{
			countryCode: "KZ",
			isDST:       false,
			name:        "Almaty Standard Time",
			offset:      21600,
			offsetHHMM:  "+06:00",
		},
	},
	"ALMST": []*TzAbbreviationInfo{
		{
			countryCode: "KZ",
			isDST:       true,
			name:        "Almaty Summer Time",
			offset:      25200,
			offsetHHMM:  "+07:00",
		},
	},
	"ANAT": []*TzAbbreviationInfo{
		{
			countryCode: "RU",
			isDST:       false,
			name:        "Anadyr Time/Anadyr Standard Time",
			offset:      43200,
			offsetHHMM:  "+12:00",
		},
	},
	"AQTT": []*TzAbbreviationInfo{
		{
			countryCode: "KZ",
			isDST:       false,
			name:        "Aqtau Time/Aqtau Standard Time",
			offset:      18000,
			offsetHHMM:  "+05:00",
		},
		{
			countryCode: "KZ",
			isDST:       false,
			name:        "Aqtobe Time/Aqtobe Standard Time",
			offset:      18000,
			offsetHHMM:  "+05:00",
		},
	},
	"AQTST": []*TzAbbreviationInfo{
		{
			countryCode: "KZ",
			isDST:       true,
			name:        "Aqtobe Summer Time",
			offset:      21600,
			offsetHHMM:  "+06:00",
		},
	},
	"TMT": []*TzAbbreviationInfo{
		{
			countryCode: "TM",
			isDST:       false,
			name:        "Turkmenistan Time/Turkmenistan Standard Time",
			offset:      18000,
			offsetHHMM:  "+05:00",
		},
	},
	"AZT": []*TzAbbreviationInfo{
		{
			countryCode: "AZ",
			isDST:       false,
			name:        "Azerbaijan Time/Azerbaijan Standard Time",
			offset:      14400,
			offsetHHMM:  "+04:00",
		},
	},
	"AZST": []*TzAbbreviationInfo{
		{
			countryCode: "AZ",
			isDST:       true,
			name:        "Azerbaijan Summer Time",
			offset:      18000,
			offsetHHMM:  "+05:00",
		},
	},
	"ICT": []*TzAbbreviationInfo{
		{
			countryCode: "TH",
			isDST:       false,
			name:        "Indochina Time",
			offset:      25200,
			offsetHHMM:  "+07:00",
		},
	},
	"KRAT": []*TzAbbreviationInfo{
		{
			countryCode: "RU",
			isDST:       false,
			name:        "Krasnoyarsk Time/Krasnoyarsk Standard Time",
			offset:      25200,
			offsetHHMM:  "+07:00",
		},
	},
	"KGT": []*TzAbbreviationInfo{
		{
			countryCode: "KG",
			isDST:       false,
			name:        "Kyrgyzstan Time",
			offset:      21600,
			offsetHHMM:  "+06:00",
		},
	},
	"BNT": []*TzAbbreviationInfo{
		{
			countryCode: "BN",
			isDST:       false,
			name:        "Brunei Darussalam Time",
			offset:      28800,
			offsetHHMM:  "+08:00",
		},
	},
	"IST": []*TzAbbreviationInfo{
		{
			countryCode: "IN",
			isDST:       false,
			name:        "India Standard Time",
			offset:      19800,
			offsetHHMM:  "+05:30",
		},
		{
			countryCode: "IN",
			isDST:       true,
			name:        "India Summer Time",
			offset:      23400,
			offsetHHMM:  "+06:30",
		},
		{
			countryCode: "IL",
			isDST:       false,
			name:        "Israel Time/Israel Standard Time",
			offset:      7200,
			offsetHHMM:  "+02:00",
		},
		{
			countryCode: "IE",
			isDST:       true,
			name:        "Irish Standard Time",
			offset:      3600,
			offsetHHMM:  "+01:00",
		},
	},
	"YAKT": []*TzAbbreviationInfo{
		{
			countryCode: "RU",
			isDST:       false,
			name:        "Yakutsk Time/Yakutsk Standard Time",
			offset:      32400,
			offsetHHMM:  "+09:00",
		},
	},
	"YAKST": []*TzAbbreviationInfo{
		{
			countryCode: "RU",
			isDST:       true,
			name:        "Yakutsk Summer Time",
			offset:      36000,
			offsetHHMM:  "+10:00",
		},
	},
	"CHOT": []*TzAbbreviationInfo{
		{
			countryCode: "MN",
			isDST:       false,
			name:        "Choibalsan Time/Choibalsan Standard Time",
			offset:      28800,
			offsetHHMM:  "+08:00",
		},
	},
	"CHOST": []*TzAbbreviationInfo{
		{
			countryCode: "MN",
			isDST:       true,
			name:        "Choibalsan Summer Time",
			offset:      32400,
			offsetHHMM:  "+09:00",
		},
	},
	"BDT": []*TzAbbreviationInfo{
		{
			countryCode: "BD",
			isDST:       false,
			name:        "Bangladesh Time/Bangladesh Standard Time",
			offset:      21600,
			offsetHHMM:  "+06:00",
		},
	},
	"BDST": []*TzAbbreviationInfo{
		{
			countryCode: "BD",
			isDST:       true,
			name:        "Bangladesh Summer Time",
			offset:      25200,
			offsetHHMM:  "+07:00",
		},
	},
	"TLT": []*TzAbbreviationInfo{
		{
			countryCode: "TL",
			isDST:       false,
			name:        "East Timor Time",
			offset:      32400,
			offsetHHMM:  "+09:00",
		},
	},
	"GST": []*TzAbbreviationInfo{
		{
			countryCode: "AE",
			isDST:       false,
			name:        "Gulf Standard Time",
			offset:      14400,
			offsetHHMM:  "+04:00",
		},
		{
			countryCode: "GS",
			isDST:       false,
			name:        "South Georgia Time",
			offset:      -7200,
			offsetHHMM:  "-02:00",
		},
	},
	"TJT": []*TzAbbreviationInfo{
		{
			countryCode: "TJ",
			isDST:       false,
			name:        "Tajikistan Time",
			offset:      18000,
			offsetHHMM:  "+05:00",
		},
	},
	"TSD": []*TzAbbreviationInfo{
		{
			countryCode: "TJ",
			isDST:       true,
			name:        "Tashkent Summer Time",
			offset:      21600,
			offsetHHMM:  "+06:00",
		},
	},
	"HKT": []*TzAbbreviationInfo{
		{
			countryCode: "HK",
			isDST:       false,
			name:        "Hong Kong Time/Hong Kong Standard Time",
			offset:      28800,
			offsetHHMM:  "+08:00",
		},
	},
	"HKST": []*TzAbbreviationInfo{
		{
			countryCode: "HK",
			isDST:       true,
			name:        "Hong Kong Summer Time",
			offset:      32400,
			offsetHHMM:  "+09:00",
		},
	},
	"HOVT": []*TzAbbreviationInfo{
		{
			countryCode: "MN",
			isDST:       false,
			name:        "Hovd Time/Hovd Standard Time",
			offset:      25200,
			offsetHHMM:  "+07:00",
		},
	},
	"HOVST": []*TzAbbreviationInfo{
		{
			countryCode: "MN",
			isDST:       true,
			name:        "Hovd Summer Time",
			offset:      28800,
			offsetHHMM:  "+08:00",
		},
	},
	"IRKT": []*TzAbbreviationInfo{
		{
			countryCode: "RU",
			isDST:       false,
			name:        "Irkutsk Time/Irkutsk Standard Time",
			offset:      28800,
			offsetHHMM:  "+08:00",
		},
	},
	"IRKST": []*TzAbbreviationInfo{
		{
			countryCode: "RU",
			isDST:       true,
			name:        "Irkutsk Summer Time",
			offset:      32400,
			offsetHHMM:  "+09:00",
		},
	},
	"TRT": []*TzAbbreviationInfo{
		{
			countryCode: "TR",
			isDST:       false,
			name:        "Turkey Time",
			offset:      10800,
			offsetHHMM:  "+03:00",
		},
	},
	"WIB": []*TzAbbreviationInfo{
		{
			countryCode: "ID",
			isDST:       false,
			name:        "Western Indonesia Time",
			offset:      25200,
			offsetHHMM:  "+07:00",
		},
	},
	"WIT": []*TzAbbreviationInfo{
		{
			countryCode: "ID",
			isDST:       false,
			name:        "Eastern Indonesia Time",
			offset:      32400,
			offsetHHMM:  "+09:00",
		},
	},
	"IDT": []*TzAbbreviationInfo{
		{
			countryCode: "IL",
			isDST:       true,
			name:        "Israel Daylight Time",
			offset:      10800,
			offsetHHMM:  "+03:00",
		},
	},
	"AFT": []*TzAbbreviationInfo{
		{
			countryCode: "AF",
			isDST:       false,
			name:        "Afghanistan Time",
			offset:      16200,
			offsetHHMM:  "+04:30",
		},
	},
	"PETT": []*TzAbbreviationInfo{
		{
			countryCode: "RU",
			isDST:       false,
			name:        "Petropavlovsk-Kamchatski Time/Petropavlovsk-Kamchatski Standard Time",
			offset:      43200,
			offsetHHMM:  "+12:00",
		},
	},
	"PKT": []*TzAbbreviationInfo{
		{
			countryCode: "PK",
			isDST:       false,
			name:        "Pakistan Time/Pakistan Standard Time",
			offset:      18000,
			offsetHHMM:  "+05:00",
		},
	},
	"PKST": []*TzAbbreviationInfo{
		{
			countryCode: "PK",
			isDST:       true,
			name:        "Pakistan Summer Time",
			offset:      21600,
			offsetHHMM:  "+06:00",
		},
	},
	"NPT": []*TzAbbreviationInfo{
		{
			countryCode: "NP",
			isDST:       false,
			name:        "Nepal Time",
			offset:      20700,
			offsetHHMM:  "+05:45",
		},
	},
	"KRAST": []*TzAbbreviationInfo{
		{
			countryCode: "RU",
			isDST:       true,
			name:        "Krasnoyarsk Summer Time",
			offset:      28800,
			offsetHHMM:  "+08:00",
		},
	},
	"MYT": []*TzAbbreviationInfo{
		{
			countryCode: "MY",
			isDST:       false,
			name:        "Malaysia Time",
			offset:      28800,
			offsetHHMM:  "+08:00",
		},
	},
	"MLAST": []*TzAbbreviationInfo{
		{
			countryCode: "MY",
			isDST:       true,
			name:        "Malaya Summer Time",
			offset:      26400,
			offsetHHMM:  "+07:20",
		},
	},
	"BORTST": []*TzAbbreviationInfo{
		{
			countryCode: "MY",
			isDST:       true,
			name:        "Borneo Summer Time",
			offset:      30000,
			offsetHHMM:  "+08:20",
		},
	},
	"MAGT": []*TzAbbreviationInfo{
		{
			countryCode: "RU",
			isDST:       false,
			name:        "Magadan Time/Magadan Standard Time",
			offset:      39600,
			offsetHHMM:  "+11:00",
		},
	},
	"MAGST": []*TzAbbreviationInfo{
		{
			countryCode: "RU",
			isDST:       true,
			name:        "Magadan Summer Time",
			offset:      43200,
			offsetHHMM:  "+12:00",
		},
	},
	"WITA": []*TzAbbreviationInfo{
		{
			countryCode: "ID",
			isDST:       false,
			name:        "Central Indonesia Time",
			offset:      28800,
			offsetHHMM:  "+08:00",
		},
	},
	"PHT": []*TzAbbreviationInfo{
		{
			countryCode: "PH",
			isDST:       false,
			name:        "Philippine Time/Philippine Standard Time",
			offset:      28800,
			offsetHHMM:  "+08:00",
		},
	},
	"PHST": []*TzAbbreviationInfo{
		{
			countryCode: "PH",
			isDST:       true,
			name:        "Philippine Summer Time",
			offset:      32400,
			offsetHHMM:  "+09:00",
		},
	},
	"NOVT": []*TzAbbreviationInfo{
		{
			countryCode: "RU",
			isDST:       false,
			name:        "Novosibirsk Time/Novosibirsk Standard Time",
			offset:      25200,
			offsetHHMM:  "+07:00",
		},
	},
	"OMST": []*TzAbbreviationInfo{
		{
			countryCode: "RU",
			isDST:       false,
			name:        "Omsk Time/Omsk Standard Time",
			offset:      21600,
			offsetHHMM:  "+06:00",
		},
	},
	"OMSST": []*TzAbbreviationInfo{
		{
			countryCode: "RU",
			isDST:       true,
			name:        "Omsk Summer Time",
			offset:      25200,
			offsetHHMM:  "+07:00",
		},
	},
	"ORAT": []*TzAbbreviationInfo{
		{
			countryCode: "KZ",
			isDST:       false,
			name:        "Oral Time",
			offset:      18000,
			offsetHHMM:  "+05:00",
		},
	},
	"KT": []*TzAbbreviationInfo{
		{
			countryCode: "KP",
			isDST:       false,
			name:        "Korean Time",
			offset:      32400,
			offsetHHMM:  "+09:00",
		},
	},
	"KST": []*TzAbbreviationInfo{
		{
			countryCode: "KP",
			isDST:       false,
			name:        "Korean Standard Time",
			offset:      32400,
			offsetHHMM:  "+09:00",
		},
	},
	"QYZT": []*TzAbbreviationInfo{
		{
			countryCode: "KZ",
			isDST:       false,
			name:        "Qyzylorda Time/Qyzylorda Standard Time",
			offset:      18000,
			offsetHHMM:  "+05:00",
		},
	},
	"QYZST": []*TzAbbreviationInfo{
		{
			countryCode: "KZ",
			isDST:       true,
			name:        "Qyzylorda Summer Time",
			offset:      21600,
			offsetHHMM:  "+06:00",
		},
	},
	"MMT": []*TzAbbreviationInfo{
		{
			countryCode: "MM",
			isDST:       false,
			name:        "Myanmar Time",
			offset:      23400,
			offsetHHMM:  "+06:30",
		},
	},
	"SAKT": []*TzAbbreviationInfo{
		{
			countryCode: "RU",
			isDST:       false,
			name:        "Sakhalin Time/Sakhalin Standard Time",
			offset:      39600,
			offsetHHMM:  "+11:00",
		},
	},
	"UZT": []*TzAbbreviationInfo{
		{
			countryCode: "UZ",
			isDST:       false,
			name:        "Uzbekistan Time/Uzbekistan Standard Time",
			offset:      18000,
			offsetHHMM:  "+05:00",
		},
	},
	"UZST": []*TzAbbreviationInfo{
		{
			countryCode: "UZ",
			isDST:       true,
			name:        "Uzbekistan Summer Time",
			offset:      21600,
			offsetHHMM:  "+06:00",
		},
	},
	"KDT": []*TzAbbreviationInfo{
		{
			countryCode: "KR",
			isDST:       true,
			name:        "Korean Daylight Time",
			offset:      36000,
			offsetHHMM:  "+10:00",
		},
	},
	"SGT": []*TzAbbreviationInfo{
		{
			countryCode: "SG",
			isDST:       false,
			name:        "Singapore Standard Time",
			offset:      28800,
			offsetHHMM:  "+08:00",
		},
	},
	"MALST": []*TzAbbreviationInfo{
		{
			countryCode: "SG",
			isDST:       true,
			name:        "Malaya Summer Time",
			offset:      26400,
			offsetHHMM:  "+07:20",
		},
	},
	"SRET": []*TzAbbreviationInfo{
		{
			countryCode: "RU",
			isDST:       false,
			name:        "Srednekolymsk Time",
			offset:      39600,
			offsetHHMM:  "+11:00",
		},
	},
	"GET": []*TzAbbreviationInfo{
		{
			countryCode: "GE",
			isDST:       false,
			name:        "Georgia Time/Georgia Standard Time",
			offset:      14400,
			offsetHHMM:  "+04:00",
		},
	},
	"IRST": []*TzAbbreviationInfo{
		{
			countryCode: "IR",
			isDST:       false,
			name:        "Iran Time/Iran Standard Time",
			offset:      12600,
			offsetHHMM:  "+03:30",
		},
	},
	"IRDT": []*TzAbbreviationInfo{
		{
			countryCode: "IR",
			isDST:       true,
			name:        "Iran Daylight Time",
			offset:      16200,
			offsetHHMM:  "+04:30",
		},
	},
	"BTT": []*TzAbbreviationInfo{
		{
			countryCode: "BT",
			isDST:       false,
			name:        "Bhutan Time",
			offset:      21600,
			offsetHHMM:  "+06:00",
		},
	},
	"JST": []*TzAbbreviationInfo{
		{
			countryCode: "JP",
			isDST:       false,
			name:        "Japan Time/Japan Standard Time",
			offset:      32400,
			offsetHHMM:  "+09:00",
		},
	},
	"JDT": []*TzAbbreviationInfo{
		{
			countryCode: "JP",
			isDST:       true,
			name:        "Japan Daylight Time",
			offset:      39600,
			offsetHHMM:  "+11:00",
		},
	},
	"ULAT": []*TzAbbreviationInfo{
		{
			countryCode: "MN",
			isDST:       false,
			name:        "Ulaanbaatar Time/Ulaanbaatar Standard Time",
			offset:      28800,
			offsetHHMM:  "+08:00",
		},
	},
	"ULAST": []*TzAbbreviationInfo{
		{
			countryCode: "MN",
			isDST:       true,
			name:        "Ulaanbaatar Summer Time",
			offset:      32400,
			offsetHHMM:  "+09:00",
		},
	},
	"VLAT": []*TzAbbreviationInfo{
		{
			countryCode: "RU",
			isDST:       false,
			name:        "Vladivostok Time/Vladivostok Standard Time",
			offset:      36000,
			offsetHHMM:  "+10:00",
		},
	},
	"VLAST": []*TzAbbreviationInfo{
		{
			countryCode: "RU",
			isDST:       true,
			name:        "Vladivostok Summer Time",
			offset:      43200,
			offsetHHMM:  "+12:00",
		},
	},
	"YEKT": []*TzAbbreviationInfo{
		{
			countryCode: "RU",
			isDST:       false,
			name:        "Yekaterinburg Time/Yekaterinburg Standard Time",
			offset:      18000,
			offsetHHMM:  "+05:00",
		},
	},
	"YEKST": []*TzAbbreviationInfo{
		{
			countryCode: "RU",
			isDST:       true,
			name:        "Yekaterinburg Summer Time",
			offset:      21600,
			offsetHHMM:  "+06:00",
		},
	},
	"AZOT": []*TzAbbreviationInfo{
		{
			countryCode: "PT",
			isDST:       false,
			name:        "Azores Time/Azores Standard Time",
			offset:      -3600,
			offsetHHMM:  "-01:00",
		},
	},
	"AZOST": []*TzAbbreviationInfo{
		{
			countryCode: "PT",
			isDST:       true,
			name:        "Azores Summer Time",
			offset:      0,
			offsetHHMM:  "+00:00",
		},
	},
	"CVT": []*TzAbbreviationInfo{
		{
			countryCode: "CV",
			isDST:       false,
			name:        "Cape Verde Time/Cape Verde Standard Time",
			offset:      -3600,
			offsetHHMM:  "-01:00",
		},
	},
	"FKT": []*TzAbbreviationInfo{
		{
			countryCode: "FK",
			isDST:       false,
			name:        "Falkland Islands Time/Falkland Islands Standard Time",
			offset:      -10800,
			offsetHHMM:  "-03:00",
		},
	},
	"AET": []*TzAbbreviationInfo{
		{
			countryCode: "AU",
			isDST:       false,
			name:        "Eastern Australia Time",
			offset:      36000,
			offsetHHMM:  "+10:00",
		},
	},
	"AEST": []*TzAbbreviationInfo{
		{
			countryCode: "AU",
			isDST:       false,
			name:        "Australian Eastern Standard Time",
			offset:      36000,
			offsetHHMM:  "+10:00",
		},
	},
	"AEDT": []*TzAbbreviationInfo{
		{
			countryCode: "AU",
			isDST:       true,
			name:        "Australian Eastern Daylight Time",
			offset:      39600,
			offsetHHMM:  "+11:00",
		},
	},
	"ACDT": []*TzAbbreviationInfo{
		{
			countryCode: "AU",
			isDST:       true,
			name:        "Australian Central Daylight Time",
			offset:      37800,
			offsetHHMM:  "+10:30",
		},
	},
	"ACWT": []*TzAbbreviationInfo{
		{
			countryCode: "AU",
			isDST:       false,
			name:        "Australian Central Western Time",
			offset:      31500,
			offsetHHMM:  "+08:45",
		},
	},
	"ACWST": []*TzAbbreviationInfo{
		{
			countryCode: "AU",
			isDST:       false,
			name:        "Australian Central Western Standard Time",
			offset:      31500,
			offsetHHMM:  "+08:45",
		},
	},
	"ACWDT": []*TzAbbreviationInfo{
		{
			countryCode: "AU",
			isDST:       true,
			name:        "Australian Central Western Daylight Time",
			offset:      35100,
			offsetHHMM:  "+09:45",
		},
	},
	"LHT": []*TzAbbreviationInfo{
		{
			countryCode: "AU",
			isDST:       false,
			name:        "Lord Howe Time",
			offset:      37800,
			offsetHHMM:  "+10:30",
		},
	},
	"LHST": []*TzAbbreviationInfo{
		{
			countryCode: "AU",
			isDST:       false,
			name:        "Lord Howe Standard Time",
			offset:      37800,
			offsetHHMM:  "+10:30",
		},
	},
	"LHDT": []*TzAbbreviationInfo{
		{
			countryCode: "AU",
			isDST:       true,
			name:        "Lord Howe Daylight Time",
			offset:      39600,
			offsetHHMM:  "+11:00",
		},
	},
	"AWDT": []*TzAbbreviationInfo{
		{
			countryCode: "AU",
			isDST:       true,
			name:        "Australian Western Daylight Time",
			offset:      32400,
			offsetHHMM:  "+09:00",
		},
	},
	"EAST": []*TzAbbreviationInfo{
		{
			countryCode: "CL",
			isDST:       false,
			name:        "Easter Island Time/Easter Island Standard Time",
			offset:      -21600,
			offsetHHMM:  "-06:00",
		},
	},
	"EASST": []*TzAbbreviationInfo{
		{
			countryCode: "CL",
			isDST:       true,
			name:        "Easter Island Summer Time",
			offset:      -18000,
			offsetHHMM:  "-05:00",
		},
	},
	"GMT-1": []*TzAbbreviationInfo{
		{
			countryCode: "",
			isDST:       false,
			name:        "Greenwich Mean Time -1",
			offset:      -3600,
			offsetHHMM:  "-01:00",
		},
	},
	"GMT-10": []*TzAbbreviationInfo{
		{
			countryCode: "",
			isDST:       false,
			name:        "Greenwich Mean Time -10",
			offset:      -36000,
			offsetHHMM:  "-10:00",
		},
	},
	"GMT-11": []*TzAbbreviationInfo{
		{
			countryCode: "",
			isDST:       false,
			name:        "Greenwich Mean Time -11",
			offset:      -39600,
			offsetHHMM:  "-11:00",
		},
	},
	"GMT-12": []*TzAbbreviationInfo{
		{
			countryCode: "",
			isDST:       false,
			name:        "Greenwich Mean Time -12",
			offset:      -43200,
			offsetHHMM:  "-12:00",
		},
	},
	"GMT-2": []*TzAbbreviationInfo{
		{
			countryCode: "",
			isDST:       false,
			name:        "Greenwich Mean Time -2",
			offset:      -7200,
			offsetHHMM:  "-02:00",
		},
	},
	"GMT-3": []*TzAbbreviationInfo{
		{
			countryCode: "",
			isDST:       false,
			name:        "Greenwich Mean Time -3",
			offset:      -10800,
			offsetHHMM:  "-03:00",
		},
	},
	"GMT-4": []*TzAbbreviationInfo{
		{
			countryCode: "",
			isDST:       false,
			name:        "Greenwich Mean Time -4",
			offset:      -14400,
			offsetHHMM:  "-04:00",
		},
	},
	"GMT-5": []*TzAbbreviationInfo{
		{
			countryCode: "",
			isDST:       false,
			name:        "Greenwich Mean Time -5",
			offset:      -18000,
			offsetHHMM:  "-05:00",
		},
	},
	"GMT-6": []*TzAbbreviationInfo{
		{
			countryCode: "",
			isDST:       false,
			name:        "Greenwich Mean Time -6",
			offset:      -21600,
			offsetHHMM:  "-06:00",
		},
	},
	"GMT-7": []*TzAbbreviationInfo{
		{
			countryCode: "",
			isDST:       false,
			name:        "Greenwich Mean Time -7",
			offset:      -25200,
			offsetHHMM:  "-07:00",
		},
	},
	"GMT-8": []*TzAbbreviationInfo{
		{
			countryCode: "",
			isDST:       false,
			name:        "Greenwich Mean Time -8",
			offset:      -28800,
			offsetHHMM:  "-08:00",
		},
	},
	"GMT-9": []*TzAbbreviationInfo{
		{
			countryCode: "",
			isDST:       false,
			name:        "Greenwich Mean Time -9",
			offset:      -32400,
			offsetHHMM:  "-09:00",
		},
	},
	"GMT+1": []*TzAbbreviationInfo{
		{
			countryCode: "",
			isDST:       false,
			name:        "Greenwich Mean Time +1",
			offset:      3600,
			offsetHHMM:  "+01:00",
		},
	},
	"GMT+10": []*TzAbbreviationInfo{
		{
			countryCode: "",
			isDST:       false,
			name:        "Greenwich Mean Time +10",
			offset:      36000,
			offsetHHMM:  "+10:00",
		},
	},
	"GMT+11": []*TzAbbreviationInfo{
		{
			countryCode: "",
			isDST:       false,
			name:        "Greenwich Mean Time +11",
			offset:      39600,
			offsetHHMM:  "+11:00",
		},
	},
	"GMT+12": []*TzAbbreviationInfo{
		{
			countryCode: "",
			isDST:       false,
			name:        "Greenwich Mean Time +12",
			offset:      43200,
			offsetHHMM:  "+12:00",
		},
	},
	"GMT+13": []*TzAbbreviationInfo{
		{
			countryCode: "",
			isDST:       false,
			name:        "Greenwich Mean Time +13",
			offset:      46800,
			offsetHHMM:  "+13:00",
		},
	},
	"GMT+14": []*TzAbbreviationInfo{
		{
			countryCode: "",
			isDST:       false,
			name:        "Greenwich Mean Time +14",
			offset:      50400,
			offsetHHMM:  "+14:00",
		},
	},
	"GMT+2": []*TzAbbreviationInfo{
		{
			countryCode: "",
			isDST:       false,
			name:        "Greenwich Mean Time +2",
			offset:      7200,
			offsetHHMM:  "+02:00",
		},
	},
	"GMT+3": []*TzAbbreviationInfo{
		{
			countryCode: "",
			isDST:       false,
			name:        "Greenwich Mean Time +3",
			offset:      10800,
			offsetHHMM:  "+03:00",
		},
	},
	"GMT+4": []*TzAbbreviationInfo{
		{
			countryCode: "",
			isDST:       false,
			name:        "Greenwich Mean Time +4",
			offset:      14400,
			offsetHHMM:  "+04:00",
		},
	},
	"GMT+5": []*TzAbbreviationInfo{
		{
			countryCode: "",
			isDST:       false,
			name:        "Greenwich Mean Time +5",
			offset:      18000,
			offsetHHMM:  "+05:00",
		},
	},
	"GMT+6": []*TzAbbreviationInfo{
		{
			countryCode: "",
			isDST:       false,
			name:        "Greenwich Mean Time +6",
			offset:      21600,
			offsetHHMM:  "+06:00",
		},
	},
	"GMT+7": []*TzAbbreviationInfo{
		{
			countryCode: "",
			isDST:       false,
			name:        "Greenwich Mean Time +7",
			offset:      25200,
			offsetHHMM:  "+07:00",
		},
	},
	"GMT+8": []*TzAbbreviationInfo{
		{
			countryCode: "",
			isDST:       false,
			name:        "Greenwich Mean Time +8",
			offset:      28800,
			offsetHHMM:  "+08:00",
		},
	},
	"GMT+9": []*TzAbbreviationInfo{
		{
			countryCode: "",
			isDST:       false,
			name:        "Greenwich Mean Time +9",
			offset:      32400,
			offsetHHMM:  "+09:00",
		},
	},
	"UTC": []*TzAbbreviationInfo{
		{
			countryCode: "",
			isDST:       false,
			name:        "Coordinated Universal Time",
			offset:      0,
			offsetHHMM:  "+00:00",
		},
	},
	"SAMT": []*TzAbbreviationInfo{
		{
			countryCode: "RU",
			isDST:       false,
			name:        "Samara Time/Samara Standard Time",
			offset:      14400,
			offsetHHMM:  "+04:00",
		},
		{
			countryCode: "RU",
			isDST:       false,
			name:        "Samara Standard Time",
			offset:      14400,
			offsetHHMM:  "+04:00",
		},
	},
	"MSK": []*TzAbbreviationInfo{
		{
			countryCode: "RU",
			isDST:       false,
			name:        "Moscow Time/Moscow Standard Time",
			offset:      10800,
			offsetHHMM:  "+03:00",
		},
	},
	"MSD": []*TzAbbreviationInfo{
		{
			countryCode: "RU",
			isDST:       true,
			name:        "Moscow Summer Time",
			offset:      14400,
			offsetHHMM:  "+04:00",
		},
	},
	"GMT+04:00": []*TzAbbreviationInfo{
		{
			countryCode: "RU",
			isDST:       false,
			name:        "Saratov Standard Time",
			offset:      14400,
			offsetHHMM:  "+04:00",
		},
	},
	"VOLT": []*TzAbbreviationInfo{
		{
			countryCode: "RU",
			isDST:       false,
			name:        "Volgograd Time/Volgograd Standard Time",
			offset:      14400,
			offsetHHMM:  "+04:00",
		},
	},
	"-00": []*TzAbbreviationInfo{
		{
			countryCode: "GB",
			isDST:       false,
			name:        "Undefined",
			offset:      0,
			offsetHHMM:  "+00:00",
		},
	},
	"IOT": []*TzAbbreviationInfo{
		{
			countryCode: "IO",
			isDST:       false,
			name:        "Indian Ocean Time",
			offset:      21600,
			offsetHHMM:  "+06:00",
		},
	},
	"CXT": []*TzAbbreviationInfo{
		{
			countryCode: "CX",
			isDST:       false,
			name:        "Christmas Island Time",
			offset:      25200,
			offsetHHMM:  "+07:00",
		},
	},
	"CCT": []*TzAbbreviationInfo{
		{
			countryCode: "CC",
			isDST:       false,
			name:        "Cocos Islands Time",
			offset:      23400,
			offsetHHMM:  "+06:30",
		},
	},
	"TFT": []*TzAbbreviationInfo{
		{
			countryCode: "TF",
			isDST:       false,
			name:        "French Southern & Antarctic Time",
			offset:      18000,
			offsetHHMM:  "+05:00",
		},
	},
	"SCT": []*TzAbbreviationInfo{
		{
			countryCode: "SC",
			isDST:       false,
			name:        "Seychelles Time",
			offset:      14400,
			offsetHHMM:  "+04:00",
		},
	},
	"MVT": []*TzAbbreviationInfo{
		{
			countryCode: "MV",
			isDST:       false,
			name:        "Maldives Time",
			offset:      18000,
			offsetHHMM:  "+05:00",
		},
	},
	"MUT": []*TzAbbreviationInfo{
		{
			countryCode: "MU",
			isDST:       false,
			name:        "Mauritius Time/Mauritius Standard Time",
			offset:      14400,
			offsetHHMM:  "+04:00",
		},
	},
	"MUST": []*TzAbbreviationInfo{
		{
			countryCode: "MU",
			isDST:       true,
			name:        "Mauritius Summer Time",
			offset:      18000,
			offsetHHMM:  "+05:00",
		},
	},
	"RET": []*TzAbbreviationInfo{
		{
			countryCode: "RE",
			isDST:       false,
			name:        "Réunion Time",
			offset:      14400,
			offsetHHMM:  "+04:00",
		},
	},
	"IRT": []*TzAbbreviationInfo{
		{
			countryCode: "IR",
			isDST:       false,
			name:        "Iran Time",
			offset:      12600,
			offsetHHMM:  "+03:30",
		},
	},
	"MHT": []*TzAbbreviationInfo{
		{
			countryCode: "MH",
			isDST:       false,
			name:        "Marshall Islands Time",
			offset:      43200,
			offsetHHMM:  "+12:00",
		},
	},
	"MET": []*TzAbbreviationInfo{
		{
			countryCode: "",
			isDST:       false,
			name:        "Middle European Time",
			offset:      3600,
			offsetHHMM:  "+01:00",
		},
	},
	"MEST": []*TzAbbreviationInfo{
		{
			countryCode: "",
			isDST:       true,
			name:        "Middle European Summer Time",
			offset:      7200,
			offsetHHMM:  "+02:00",
		},
	},
	"CHAT": []*TzAbbreviationInfo{
		{
			countryCode: "NZ",
			isDST:       false,
			name:        "Chatham Time",
			offset:      45900,
			offsetHHMM:  "+12:45",
		},
	},
	"CHAST": []*TzAbbreviationInfo{
		{
			countryCode: "NZ",
			isDST:       false,
			name:        "Chatham Standard Time",
			offset:      45900,
			offsetHHMM:  "+12:45",
		},
	},
	"CHADT": []*TzAbbreviationInfo{
		{
			countryCode: "NZ",
			isDST:       true,
			name:        "Chatham Daylight Time",
			offset:      49500,
			offsetHHMM:  "+13:45",
		},
	},
	"WST": []*TzAbbreviationInfo{
		{
			countryCode: "WS",
			isDST:       false,
			name:        "Apia Time/Apia Standard Time",
			offset:      46800,
			offsetHHMM:  "+13:00",
		},
	},
	"WSDT": []*TzAbbreviationInfo{
		{
			countryCode: "WS",
			isDST:       true,
			name:        "Apia Daylight Time",
			offset:      50400,
			offsetHHMM:  "+14:00",
		},
	},
	"CHUT": []*TzAbbreviationInfo{
		{
			countryCode: "FM",
			isDST:       false,
			name:        "Chuuk Time",
			offset:      36000,
			offsetHHMM:  "+10:00",
		},
	},
	"VUT": []*TzAbbreviationInfo{
		{
			countryCode: "VU",
			isDST:       false,
			name:        "Vanuatu Time/Vanuatu Standard Time",
			offset:      39600,
			offsetHHMM:  "+11:00",
		},
	},
	"VUST": []*TzAbbreviationInfo{
		{
			countryCode: "VU",
			isDST:       true,
			name:        "Vanuatu Summer Time",
			offset:      43200,
			offsetHHMM:  "+12:00",
		},
	},
	"PHOT": []*TzAbbreviationInfo{
		{
			countryCode: "KI",
			isDST:       false,
			name:        "Phoenix Islands Time",
			offset:      46800,
			offsetHHMM:  "+13:00",
		},
	},
	"TKT": []*TzAbbreviationInfo{
		{
			countryCode: "TK",
			isDST:       false,
			name:        "Tokelau Time",
			offset:      46800,
			offsetHHMM:  "+13:00",
		},
	},
	"FJT": []*TzAbbreviationInfo{
		{
			countryCode: "FJ",
			isDST:       false,
			name:        "Fiji Time/Fiji Standard Time",
			offset:      43200,
			offsetHHMM:  "+12:00",
		},
	},
	"FJST": []*TzAbbreviationInfo{
		{
			countryCode: "FJ",
			isDST:       true,
			name:        "Fiji Summer Time",
			offset:      46800,
			offsetHHMM:  "+13:00",
		},
	},
	"TVT": []*TzAbbreviationInfo{
		{
			countryCode: "TV",
			isDST:       false,
			name:        "Tuvalu Time",
			offset:      43200,
			offsetHHMM:  "+12:00",
		},
	},
	"GALT": []*TzAbbreviationInfo{
		{
			countryCode: "EC",
			isDST:       false,
			name:        "Galapagos Time",
			offset:      -21600,
			offsetHHMM:  "-06:00",
		},
	},
	"GAMT": []*TzAbbreviationInfo{
		{
			countryCode: "PF",
			isDST:       false,
			name:        "Gambier Time",
			offset:      -32400,
			offsetHHMM:  "-09:00",
		},
	},
	"SBT": []*TzAbbreviationInfo{
		{
			countryCode: "SB",
			isDST:       false,
			name:        "Solomon Islands Time",
			offset:      39600,
			offsetHHMM:  "+11:00",
		},
	},
	"ChST": []*TzAbbreviationInfo{
		{
			countryCode: "GU",
			isDST:       false,
			name:        "Chamorro Standard Time",
			offset:      36000,
			offsetHHMM:  "+10:00",
		},
	},
	"GDT": []*TzAbbreviationInfo{
		{
			countryCode: "GU",
			isDST:       true,
			name:        "Guam Daylight Time",
			offset:      39600,
			offsetHHMM:  "+11:00",
		},
	},
	"LINT": []*TzAbbreviationInfo{
		{
			countryCode: "KI",
			isDST:       false,
			name:        "Line Islands Time",
			offset:      50400,
			offsetHHMM:  "+14:00",
		},
	},
	"KOST": []*TzAbbreviationInfo{
		{
			countryCode: "FM",
			isDST:       false,
			name:        "Kosrae Time",
			offset:      39600,
			offsetHHMM:  "+11:00",
		},
	},
	"MART": []*TzAbbreviationInfo{
		{
			countryCode: "PF",
			isDST:       false,
			name:        "Marquesas Time",
			offset:      -34200,
			offsetHHMM:  "-09:30",
		},
	},
	"SST": []*TzAbbreviationInfo{
		{
			countryCode: "UM",
			isDST:       false,
			name:        "Samoa Time/Samoa Standard Time",
			offset:      -39600,
			offsetHHMM:  "-11:00",
		},
	},
	"NRT": []*TzAbbreviationInfo{
		{
			countryCode: "NR",
			isDST:       false,
			name:        "Nauru Time",
			offset:      43200,
			offsetHHMM:  "+12:00",
		},
	},
	"NUT": []*TzAbbreviationInfo{
		{
			countryCode: "NU",
			isDST:       false,
			name:        "Niue Time",
			offset:      -39600,
			offsetHHMM:  "-11:00",
		},
	},
	"NFT": []*TzAbbreviationInfo{
		{
			countryCode: "NF",
			isDST:       false,
			name:        "Norfolk Island Time/Norfolk Island Standard Time",
			offset:      39600,
			offsetHHMM:  "+11:00",
		},
	},
	"NFDT": []*TzAbbreviationInfo{
		{
			countryCode: "NF",
			isDST:       true,
			name:        "Norfolk Island Daylight Time",
			offset:      43200,
			offsetHHMM:  "+12:00",
		},
	},
	"NCT": []*TzAbbreviationInfo{
		{
			countryCode: "NC",
			isDST:       false,
			name:        "New Caledonia Time/New Caledonia Standard Time",
			offset:      39600,
			offsetHHMM:  "+11:00",
		},
	},
	"NCST": []*TzAbbreviationInfo{
		{
			countryCode: "NC",
			isDST:       true,
			name:        "New Caledonia Summer Time",
			offset:      43200,
			offsetHHMM:  "+12:00",
		},
	},
	"PWT": []*TzAbbreviationInfo{
		{
			countryCode: "PW",
			isDST:       false,
			name:        "Palau Time",
			offset:      32400,
			offsetHHMM:  "+09:00",
		},
	},
	"PONT": []*TzAbbreviationInfo{
		{
			countryCode: "FM",
			isDST:       false,
			name:        "Ponape Time",
			offset:      39600,
			offsetHHMM:  "+11:00",
		},
	},
	"PGT": []*TzAbbreviationInfo{
		{
			countryCode: "PG",
			isDST:       false,
			name:        "Papua New Guinea Time",
			offset:      36000,
			offsetHHMM:  "+10:00",
		},
	},
	"CKT": []*TzAbbreviationInfo{
		{
			countryCode: "CK",
			isDST:       false,
			name:        "Cook Islands Time/Cook Islands Standard Time",
			offset:      -36000,
			offsetHHMM:  "-10:00",
		},
	},
	"CKHST": []*TzAbbreviationInfo{
		{
			countryCode: "CK",
			isDST:       true,
			name:        "Cook Islands Half Summer Time",
			offset:      -34200,
			offsetHHMM:  "-09:30",
		},
	},
	"TAHT": []*TzAbbreviationInfo{
		{
			countryCode: "PF",
			isDST:       false,
			name:        "Tahiti Time",
			offset:      -36000,
			offsetHHMM:  "-10:00",
		},
	},
	"GILT": []*TzAbbreviationInfo{
		{
			countryCode: "KI",
			isDST:       false,
			name:        "Gilbert Islands Time",
			offset:      43200,
			offsetHHMM:  "+12:00",
		},
	},
	"TOT": []*TzAbbreviationInfo{
		{
			countryCode: "TO",
			isDST:       false,
			name:        "Tonga Time/Tonga Standard Time",
			offset:      46800,
			offsetHHMM:  "+13:00",
		},
	},
	"TOST": []*TzAbbreviationInfo{
		{
			countryCode: "TO",
			isDST:       true,
			name:        "Tonga Summer Time",
			offset:      50400,
			offsetHHMM:  "+14:00",
		},
	},
	"WAKT": []*TzAbbreviationInfo{
		{
			countryCode: "UM",
			isDST:       false,
			name:        "Wake Island Time",
			offset:      43200,
			offsetHHMM:  "+12:00",
		},
	},
	"WFT": []*TzAbbreviationInfo{
		{
			countryCode: "WF",
			isDST:       false,
			name:        "Wallis & Futuna Time",
			offset:      43200,
			offsetHHMM:  "+12:00",
		},
	},
	"GMT+3:30": []*TzAbbreviationInfo{
		{
			countryCode: "",
			isDST:       false,
			name:        "Greenwich Mean Time +3:30",
			offset:      12600,
			offsetHHMM:  "+03:30",
		},
	},
	"GMT+4:30": []*TzAbbreviationInfo{
		{
			countryCode: "",
			isDST:       false,
			name:        "Greenwich Mean Time +4:30",
			offset:      16200,
			offsetHHMM:  "+04:30",
		},
	},
	"GMT+5:45": []*TzAbbreviationInfo{
		{
			countryCode: "",
			isDST:       false,
			name:        "Greenwich Mean Time +5:45",
			offset:      20700,
			offsetHHMM:  "+05:45",
		},
	},
	"GMT+6:30": []*TzAbbreviationInfo{
		{
			countryCode: "",
			isDST:       false,
			name:        "Greenwich Mean Time +6:30",
			offset:      23400,
			offsetHHMM:  "+06:30",
		},
	},
	"GMT+8:45": []*TzAbbreviationInfo{
		{
			countryCode: "",
			isDST:       false,
			name:        "Greenwich Mean Time +8:45",
			offset:      31500,
			offsetHHMM:  "+08:45",
		},
	},
	"GMT+9:30": []*TzAbbreviationInfo{
		{
			countryCode: "",
			isDST:       false,
			name:        "Greenwich Mean Time +9:30",
			offset:      34200,
			offsetHHMM:  "+09:30",
		},
	},
	"GMT+10:30": []*TzAbbreviationInfo{
		{
			countryCode: "",
			isDST:       false,
			name:        "Greenwich Mean Time +10:30",
			offset:      37800,
			offsetHHMM:  "+10:30",
		},
	},
	"GMT+13:45": []*TzAbbreviationInfo{
		{
			countryCode: "",
			isDST:       false,
			name:        "Greenwich Mean Time +13:45",
			offset:      49500,
			offsetHHMM:  "+13:45",
		},
	},
	"GMT-9:30": []*TzAbbreviationInfo{
		{
			countryCode: "",
			isDST:       false,
			name:        "Greenwich Mean Time -9:30",
			offset:      -34200,
			offsetHHMM:  "-09:30",
		},
	},
	// military timezones
	"A": []*TzAbbreviationInfo{
		{
			name:       "Alfa Time Zone",
			offset:     3600,
			offsetHHMM: "+01:00",
		},
	},
	"B": []*TzAbbreviationInfo{
		{
			name:       "Bravo Time Zone",
			offset:     7200,
			offsetHHMM: "+02:00",
		},
	},
	"C": []*TzAbbreviationInfo{
		{
			name:       "Charlie Time Zone",
			offset:     10800,
			offsetHHMM: "+03:00",
		},
	},
	"D": []*TzAbbreviationInfo{
		{
			name:       "Delta Time Zone",
			offset:     14400,
			offsetHHMM: "+04:00",
		},
	},
	"E": []*TzAbbreviationInfo{
		{
			name:       "Echo Time Zone",
			offset:     18000,
			offsetHHMM: "+05:00",
		},
	},
	"F": []*TzAbbreviationInfo{
		{
			name:       "Foxtrot Time Zone",
			offset:     21600,
			offsetHHMM: "+06:00",
		},
	},
	"G": []*TzAbbreviationInfo{
		{
			name:       "Golf Time Zone",
			offset:     25200,
			offsetHHMM: "+07:00",
		},
	},
	"H": []*TzAbbreviationInfo{
		{
			name:       "Hotel Time Zone",
			offset:     28800,
			offsetHHMM: "+08:00",
		},
	},
	"I": []*TzAbbreviationInfo{
		{
			name:       "India Time Zone",
			offset:     32400,
			offsetHHMM: "+09:00",
		},
	},
	"K": []*TzAbbreviationInfo{
		{
			name:       "Kilo Time Zone",
			offset:     36000,
			offsetHHMM: "+10:00",
		},
	},
	"L": []*TzAbbreviationInfo{
		{
			name:       "Lima Time Zone",
			offset:     39600,
			offsetHHMM: "+11:00",
		},
	},
	"M": []*TzAbbreviationInfo{
		{
			name:       "Mike Time Zone",
			offset:     43200,
			offsetHHMM: "+12:00",
		},
	},
	"N": []*TzAbbreviationInfo{
		{
			name:       "November Time Zone",
			offset:     -3600,
			offsetHHMM: "-01:00",
		},
	},
	"O": []*TzAbbreviationInfo{
		{
			name:       "Oscar Time Zone",
			offset:     -7200,
			offsetHHMM: "-02:00",
		},
	},
	"P": []*TzAbbreviationInfo{
		{
			name:       "Papa Time Zone",
			offset:     -10800,
			offsetHHMM: "-03:00",
		},
	},
	"Q": []*TzAbbreviationInfo{
		{
			name:       "Quebec Time Zone",
			offset:     -14400,
			offsetHHMM: "-04:00",
		},
	},
	"R": []*TzAbbreviationInfo{
		{
			name:       "Romeo Time Zone",
			offset:     -18000,
			offsetHHMM: "-05:00",
		},
	},
	"S": []*TzAbbreviationInfo{
		{
			name:       "Sierra Time Zone",
			offset:     -21600,
			offsetHHMM: "-06:00",
		},
	},
	"T": []*TzAbbreviationInfo{
		{
			name:       "Tango Time Zone",
			offset:     -25200,
			offsetHHMM: "-07:00",
		},
	},
	"U": []*TzAbbreviationInfo{
		{
			name:       "Uniform Time Zone",
			offset:     -28800,
			offsetHHMM: "-08:00",
		},
	},
	"V": []*TzAbbreviationInfo{
		{
			name:       "Victor Time Zone",
			offset:     -32400,
			offsetHHMM: "-09:00",
		},
	},
	"W": []*TzAbbreviationInfo{
		{
			name:       "Whiskey Time Zone",
			offset:     -36000,
			offsetHHMM: "-10:00",
		},
	},
	"X": []*TzAbbreviationInfo{
		{
			name:       "X-ray Time Zone",
			offset:     -39600,
			offsetHHMM: "-11:00",
		},
	},
	"Y": []*TzAbbreviationInfo{
		{
			name:       "Yankee Time Zone",
			offset:     -43200,
			offsetHHMM: "-12:00",
		},
	},
	"Z": []*TzAbbreviationInfo{
		{
			name:       "Zulu Time Zone",
			offset:     0,
			offsetHHMM: "+00:00",
		},
	},
}
