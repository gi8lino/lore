package main

import "strings"

type keywordGroup struct {
	class string
	words string
}

func languageDefinitions() []language {
	cLikeComments := []commentPair{{start: "/*", end: "*/"}}
	cLikeQuotes := []quoteRule{
		{start: `"`, end: `"`, escape: true, class: "s2"},
		{start: `'`, end: `'`, escape: true, class: "s1"},
	}

	return []language{
		profileLanguage("Bash", []string{"bash", "sh", "shell", "ksh", "zsh"}, codeProfile{
			lineComments: []string{"#"},
			quotes:       append(cLikeQuotes, quoteRule{start: "`", end: "`", escape: true, class: "sb"}),
			keywords: keywordMap(
				keywordGroup{"k", "if then else elif fi for while until do done case esac select in function time coproc"},
				keywordGroup{"kc", "true false"},
			),
			builtins:         wordSet("alias bg bind break builtin caller cd command compgen complete declare dirs disown echo enable eval exec exit export fc fg getopts hash help history jobs kill let local logout mapfile popd printf pushd pwd read readarray readonly set shift shopt source suspend test times trap type typeset ulimit umask unalias unset wait"),
			variablePrefixes: "$",
		}),
		profileLanguage("C", []string{"c"}, cFamilyProfile(cLikeComments, cLikeQuotes, "auto break case char const continue default do double else enum extern float for goto if inline int long register restrict return short signed sizeof static struct switch typedef union unsigned void volatile while _Bool _Complex _Imaginary", "char double float int long short signed unsigned void size_t ptrdiff_t")),
		profileLanguage("C++", []string{"cpp", "c++", "cxx", "cc", "hpp"}, cFamilyProfile(cLikeComments, cLikeQuotes, "alignas alignof and and_eq asm auto bitand bitor bool break case catch char char8_t char16_t char32_t class compl concept const consteval constexpr constinit const_cast continue co_await co_return co_yield decltype default delete do double dynamic_cast else enum explicit export extern false float for friend goto if inline int long mutable namespace new noexcept not not_eq nullptr operator or or_eq private protected public register reinterpret_cast requires return short signed sizeof static static_assert static_cast struct switch template this thread_local throw true try typedef typeid typename union unsigned using virtual void volatile wchar_t while xor xor_eq", "bool char char8_t char16_t char32_t double float int long short signed unsigned void wchar_t size_t string")),
		profileLanguage("C#", []string{"csharp", "cs", "c#"}, cFamilyProfile(cLikeComments, cLikeQuotes, "abstract as base bool break byte case catch char checked class const continue decimal default delegate do double else enum event explicit extern false finally fixed float for foreach goto if implicit in int interface internal is lock long namespace new null object operator out override params private protected public readonly record ref return sbyte sealed short sizeof stackalloc static string struct switch this throw true try typeof uint ulong unchecked unsafe ushort using virtual void volatile while async await var dynamic", "bool byte char decimal double float int long object sbyte short string uint ulong ushort void")),
		profileLanguage("CSS", []string{"css"}, codeProfile{
			blockComments:   cLikeComments,
			quotes:          cLikeQuotes,
			keywords:        keywordMap(keywordGroup{"kc", "inherit initial revert revert-layer unset transparent currentcolor"}),
			keySeparator:    ':',
			atKeyword:       true,
			identifierExtra: "-#",
		}),
		profileLanguage("Dockerfile", []string{"docker", "dockerfile", "containerfile"}, codeProfile{
			caseInsensitive:  true,
			lineComments:     []string{"#"},
			quotes:           cLikeQuotes,
			keywords:         keywordMap(keywordGroup{"k", "from run cmd label maintainer expose env add copy entrypoint volume user workdir arg onbuild stopsignal healthcheck shell as"}),
			builtins:         wordSet("echo printf cd pwd test true false set export mkdir rm cp mv chmod chown curl wget apk apt apt-get yum dnf microdnf"),
			variablePrefixes: "$",
			identifierExtra:  "-",
		}),
		profileLanguage("Go", []string{"go", "golang"}, codeProfile{
			lineComments:  []string{"//"},
			blockComments: cLikeComments,
			quotes: []quoteRule{
				{start: `"`, end: `"`, escape: true, class: "s2"},
				{start: `'`, end: `'`, escape: true, class: "s1"},
				{start: "`", end: "`", class: "sb"},
			},
			keywords: keywordMap(
				keywordGroup{"kd", "package import const type var func"},
				keywordGroup{"k", "break default select case defer go else goto switch fallthrough if range continue for return"},
				keywordGroup{"kt", "any bool byte comparable complex64 complex128 error float32 float64 int int8 int16 int32 int64 rune string uint uint8 uint16 uint32 uint64 uintptr"},
				keywordGroup{"kc", "true false nil iota"},
			),
			builtins: wordSet("append cap clear close complex copy delete imag len make max min new panic print println real recover"),
		}),
		profileLanguage("HCL", []string{"hcl", "terraform", "tf"}, codeProfile{
			lineComments:  []string{"#", "//"},
			blockComments: cLikeComments,
			quotes:        []quoteRule{{start: `"`, end: `"`, escape: true, class: "s2"}},
			keywords: keywordMap(
				keywordGroup{"kd", "terraform provider resource data variable locals output module moved import check"},
				keywordGroup{"kc", "true false null"},
				keywordGroup{"k", "for in if"},
			),
			builtins:         wordSet("abs abspath alltrue anytrue basename can ceil chomp chunklist cidrhost cidrnetmask cidrsubnet cidrsubnets coalesce coalescelist compact concat contains dirname distinct element endswith file filebase64 fileexists fileset flatten floor format formatdate formatlist indent index join jsondecode jsonencode keys length log lookup lower matchkeys max md5 merge min one parseint pathexpand plantimestamp pow range regex regexall replace reverse rsadecrypt sensitive setintersection setproduct setsubtract setunion sha1 sha256 sha512 signum slice sort split startswith strcontains substr sum textdecodebase64 textencodebase64 timestamp title tobool tolist tomap tonumber toset tostring transpose trim trimprefix trimspace trimsuffix try upper urlencode uuid uuidv5 values yamldecode yamlencode zipmap"),
			variablePrefixes: "$",
			identifierExtra:  "-",
			keySeparator:     '=',
		}),
		{name: "HTML", aliases: []string{"html", "htm"}, scan: scanMarkup},
		profileLanguage("Java", []string{"java"}, cFamilyProfile(cLikeComments, cLikeQuotes, "abstract assert boolean break byte case catch char class const continue default do double else enum extends final finally float for goto if implements import instanceof int interface long native new package private protected public return short static strictfp super switch synchronized this throw throws transient true try void volatile while false null record sealed permits non-sealed var yield", "boolean byte char double float int long short void String Object")),
		profileLanguage("JavaScript", []string{"javascript", "js", "jsx", "node"}, codeProfile{
			lineComments: []string{"//"}, blockComments: cLikeComments,
			quotes: append(cLikeQuotes, quoteRule{start: "`", end: "`", escape: true, class: "sb"}),
			keywords: keywordMap(
				keywordGroup{"kd", "const let var function class import export"},
				keywordGroup{"k", "async await break case catch continue debugger default delete do else extends finally for from get if in instanceof new of return set static super switch this throw try typeof void while with yield"},
				keywordGroup{"kc", "true false null undefined NaN Infinity"},
			),
			builtins:         wordSet("Array BigInt Boolean Date Error Function JSON Map Math Number Object Promise Proxy Reflect RegExp Set String Symbol WeakMap WeakSet console document globalThis window"),
			variablePrefixes: "$",
		}),
		profileLanguage("JSON", []string{"json", "jsonc", "jsonl"}, codeProfile{
			lineComments: []string{"//"}, blockComments: cLikeComments,
			quotes:       []quoteRule{{start: `"`, end: `"`, escape: true, class: "s2"}},
			keywords:     keywordMap(keywordGroup{"kc", "true false null"}),
			keySeparator: ':',
		}),
		profileLanguage("Kotlin", []string{"kotlin", "kt", "kts"}, cFamilyProfile(cLikeComments, cLikeQuotes, "as break class continue do else false for fun if in interface is null object package return super this throw true try typealias typeof val var when while by catch constructor delegate dynamic field file finally get import init param property receiver set setparam where actual abstract annotation companion const crossinline data enum expect external final infix inline inner internal lateinit noinline open operator out override private protected public reified sealed suspend tailrec vararg", "Any Boolean Byte Char Double Float Int Long Nothing Short String Unit UInt ULong UByte UShort")),
		profileLanguage("Lua", []string{"lua"}, codeProfile{
			lineComments: []string{"--"}, blockComments: []commentPair{{start: "--[[", end: "]]"}},
			quotes:   cLikeQuotes,
			keywords: keywordMap(keywordGroup{"k", "and break do else elseif end false for function goto if in local nil not or repeat return then true until while"}),
			builtins: wordSet("assert collectgarbage dofile error getmetatable ipairs load loadfile next pairs pcall print rawequal rawget rawlen rawset require select setmetatable tonumber tostring type xpcall"),
		}),
		profileLanguage("Makefile", []string{"make", "makefile", "mk"}, codeProfile{
			lineComments:     []string{"#"},
			quotes:           cLikeQuotes,
			keywords:         keywordMap(keywordGroup{"k", "include -include sinclude define endef ifdef ifndef ifeq ifneq else endif override export unexport private vpath"}),
			builtins:         wordSet("subst patsubst strip findstring filter filter-out sort word wordlist words firstword lastword dir notdir suffix basename addsuffix addprefix join wildcard realpath abspath error warning shell origin flavor foreach if or and intcmp call value eval file let"),
			variablePrefixes: "$",
			identifierExtra:  "-.",
			keySeparator:     ':',
		}),
		{name: "Markdown", aliases: []string{"markdown", "md", "mdown"}, scan: scanMarkdown},
		profileLanguage("Nginx", []string{"nginx"}, codeProfile{
			lineComments:     []string{"#"},
			quotes:           cLikeQuotes,
			keywords:         keywordMap(keywordGroup{"k", "http server location upstream events map geo types if limit_except include listen server_name root index proxy_pass fastcgi_pass return rewrite try_files set add_header error_page access_log error_log"}),
			variablePrefixes: "$",
			identifierExtra:  "_-.",
		}),
		profileLanguage("PHP", []string{"php"}, codeProfile{
			lineComments: []string{"//", "#"}, blockComments: cLikeComments,
			quotes: cLikeQuotes,
			keywords: keywordMap(
				keywordGroup{"kd", "class interface trait function const namespace use"},
				keywordGroup{"k", "abstract and array as break callable case catch clone continue declare default die do echo else elseif empty enddeclare endfor endforeach endif endswitch endwhile eval exit extends final finally fn for foreach global goto if implements include include_once instanceof insteadof match new or print private protected public readonly require require_once return static switch throw try unset while xor yield"},
				keywordGroup{"kc", "true false null"},
			),
			builtins:         wordSet("count strlen sprintf printf explode implode array_map array_filter array_reduce json_encode json_decode in_array isset defined header date time preg_match"),
			variablePrefixes: "$",
		}),
		profileLanguage("PowerShell", []string{"powershell", "ps1", "pwsh"}, codeProfile{
			caseInsensitive: true,
			lineComments:    []string{"#"}, blockComments: []commentPair{{start: "<#", end: "#>"}},
			quotes:           cLikeQuotes,
			keywords:         keywordMap(keywordGroup{"k", "begin break catch class continue data define do dynamicparam else elseif end enum exit filter finally for foreach from function hidden if in inlinescript parallel param process return sequence switch throw trap try until using var while workflow"}),
			builtins:         wordSet("write-host write-output write-error get-item get-childitem get-content set-content new-item remove-item copy-item move-item test-path get-command get-help where-object foreach-object select-object sort-object measure-object invoke-command invoke-restmethod invoke-webrequest"),
			variablePrefixes: "$",
			identifierExtra:  "-",
		}),
		profileLanguage("Python", []string{"python", "py", "python3"}, codeProfile{
			lineComments: []string{"#"},
			quotes: []quoteRule{
				{start: `"""`, end: `"""`, escape: true, class: "sd"},
				{start: `'''`, end: `'''`, escape: true, class: "sd"},
				{start: `"`, end: `"`, escape: true, class: "s2"},
				{start: `'`, end: `'`, escape: true, class: "s1"},
			},
			keywords: keywordMap(
				keywordGroup{"kd", "class def import from global nonlocal lambda"},
				keywordGroup{"k", "and as assert async await break case continue del elif else except finally for if in is match not or pass raise return try while with yield"},
				keywordGroup{"kc", "True False None NotImplemented Ellipsis"},
			),
			builtins:  wordSet("abs all any ascii bin bool breakpoint bytearray bytes callable chr classmethod compile complex delattr dict dir divmod enumerate eval exec filter float format frozenset getattr globals hasattr hash help hex id input int isinstance issubclass iter len list locals map max memoryview min next object oct open ord pow print property range repr reversed round set setattr slice sorted staticmethod str sum super tuple type vars zip __import__"),
			atKeyword: true,
		}),
		profileLanguage("Ruby", []string{"ruby", "rb"}, codeProfile{
			lineComments:     []string{"#"},
			quotes:           cLikeQuotes,
			keywords:         keywordMap(keywordGroup{"k", "BEGIN END alias and begin break case class def defined do else elsif end ensure false for if in module next nil not or redo rescue retry return self super then true undef unless until when while yield"}),
			builtins:         wordSet("puts print printf p require load raise fail catch throw loop proc lambda block_given? sprintf format rand sleep exit abort caller open gets readline select"),
			variablePrefixes: "$@",
			identifierExtra:  "?!",
		}),
		profileLanguage("Rust", []string{"rust", "rs"}, cFamilyProfile(cLikeComments, cLikeQuotes, "as async await break const continue crate dyn else enum extern false fn for if impl in let loop match mod move mut pub ref return self Self static struct super trait true type unsafe use where while abstract become box do final macro override priv typeof unsized virtual yield try", "bool char f32 f64 i8 i16 i32 i64 i128 isize str u8 u16 u32 u64 u128 usize")),
		profileLanguage("SQL", []string{"sql"}, codeProfile{
			caseInsensitive: true,
			lineComments:    []string{"--", "#"}, blockComments: cLikeComments,
			quotes: []quoteRule{{start: `'`, end: `'`, class: "s1"}, {start: `"`, end: `"`, class: "s2"}, {start: "`", end: "`", class: "s2"}},
			keywords: keywordMap(
				keywordGroup{"k", "add all alter analyze and any as asc begin between by case check column commit constraint create cross database default delete desc distinct drop else end except exists explain false fetch for foreign from full grant group having if in index inner insert intersect into is join key left like limit lock natural not null offset on or order outer primary references rename replace returning revoke right rollback row rows select set table then to transaction trigger true union unique update using values view when where with"},
				keywordGroup{"kt", "bigint binary bit blob boolean char date datetime decimal double enum float int integer interval json numeric real serial smallint text time timestamp tinyint uuid varchar"},
				keywordGroup{"kc", "null true false"},
			),
			builtins: wordSet("avg count max min sum coalesce concat lower upper length substring trim now current_date current_time current_timestamp extract date_trunc cast convert round floor ceil abs random"),
		}),
		profileLanguage("TOML", []string{"toml"}, codeProfile{
			lineComments: []string{"#"},
			quotes: []quoteRule{
				{start: `"""`, end: `"""`, escape: true, class: "s2"},
				{start: `'''`, end: `'''`, class: "s1"},
				{start: `"`, end: `"`, escape: true, class: "s2"},
				{start: `'`, end: `'`, class: "s1"},
			},
			keywords:        keywordMap(keywordGroup{"kc", "true false inf nan"}),
			identifierExtra: "-_",
			keySeparator:    '=',
		}),
		profileLanguage("TypeScript", []string{"typescript", "ts", "tsx"}, codeProfile{
			lineComments: []string{"//"}, blockComments: cLikeComments,
			quotes: append(cLikeQuotes, quoteRule{start: "`", end: "`", escape: true, class: "sb"}),
			keywords: keywordMap(
				keywordGroup{"kd", "abstract class const constructor declare enum export function import interface let namespace private protected public readonly static type var"},
				keywordGroup{"k", "any as asserts async await break case catch continue debugger default delete do else extends false finally for from get if implements in infer instanceof is keyof module never new null number object of override package require return satisfies set string super switch symbol this throw true try typeof undefined unique unknown void while with yield"},
				keywordGroup{"kt", "bigint boolean number object string symbol unknown never void any"},
				keywordGroup{"kc", "true false null undefined NaN Infinity"},
			),
			builtins:         wordSet("Array BigInt Boolean Date Error Function JSON Map Math Number Object Promise Proxy Reflect RegExp Set String Symbol WeakMap WeakSet console document globalThis window"),
			variablePrefixes: "$",
		}),
		{name: "XML", aliases: []string{"xml", "xhtml", "svg"}, scan: scanMarkup},
		profileLanguage("YAML", []string{"yaml", "yml"}, codeProfile{
			lineComments:    []string{"#"},
			quotes:          cLikeQuotes,
			keywords:        keywordMap(keywordGroup{"kc", "true false null yes no on off True False Null Yes No On Off TRUE FALSE NULL YES NO ON OFF"}),
			identifierExtra: "-_./",
			keySeparator:    ':',
		}),
		profileLanguage("Zig", []string{"zig"}, cFamilyProfile(nil, cLikeQuotes, "addrspace align allowzero and anyframe anytype asm async await break callconv catch comptime const continue defer else enum errdefer error export extern false fn for if inline linksection noalias noinline nosuspend null opaque or orelse packed pub resume return struct suspend switch test threadlocal true try union unreachable usingnamespace var volatile while", "bool comptime_float comptime_int f16 f32 f64 f80 f128 i8 i16 i32 i64 i128 isize noreturn type u8 u16 u32 u64 u128 usize void anyopaque")),
	}
}

func profileLanguage(name string, aliases []string, profile codeProfile) language {
	return language{name: name, aliases: aliases, scan: func(source string) []token { return scanCode(source, profile) }}
}

func cFamilyProfile(comments []commentPair, quotes []quoteRule, keywords, types string) codeProfile {
	return codeProfile{
		lineComments:  []string{"//"},
		blockComments: comments,
		quotes:        quotes,
		keywords: keywordMap(
			keywordGroup{"k", keywords},
			keywordGroup{"kt", types},
			keywordGroup{"kc", "true false null nil nullptr"},
		),
	}
}

func keywordMap(groups ...keywordGroup) map[string]string {
	result := make(map[string]string)
	for _, group := range groups {
		for word := range strings.FieldsSeq(group.words) {
			result[word] = group.class
		}
	}
	return result
}

func wordSet(words string) map[string]struct{} {
	result := make(map[string]struct{})
	for word := range strings.FieldsSeq(words) {
		result[word] = struct{}{}
	}
	return result
}
