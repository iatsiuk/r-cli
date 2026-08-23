package parser

import (
	"strings"
	"testing"
)

// productionLogCase is one expression recorded in the parser error log, labelled
// with the log entry numbers it was recorded under.
type productionLogCase struct {
	name  string
	entry string
	expr  string
}

// productionLogAccepted holds every distinct expression from the parser error log
// that the parser is now expected to accept, grouped by the root cause it exercises.
// Kept at package level so the table does not count against the function length limit.
var productionLogAccepted = []productionLogCase{
	// root cause 1: group() with anything but one string literal
	{
		name:  "group_two_string_keys",
		entry: "11",
		expr:  `r.table("products").filter({userId:"378684ed-c5ce-4b88-b793-b993780c66d4"}).group("game", "category").count().ungroup()`,
	},
	{
		name:  "group_three_string_keys",
		entry: "21, 24",
		expr:  `r.db("restored").table("products").filter({published:true}).group("game","category","locale").count().ungroup().orderBy(r.desc("reduction")).limit(8)`,
	},
	{
		name:  "group_two_keys_commissions",
		entry: "37",
		expr:  `r.db("restored").table("commissions").group("type","primary").count()`,
	},
	{
		name:  "group_two_keys_orders",
		entry: "63",
		expr:  `r.db("restored").table("orders").limit(2000).group("channel","productType").count()`,
	},
	{
		name:  "group_three_keys_after_eq_join",
		entry: "33",
		expr:  `r.table("transactions").filter({type: "ACCOUNT_REFUND"}).eqJoin("refundedTransaction", r.table("transactions")).map(function(row){ return {origType: row("right")("type"), origPurpose: row("right")("purpose").default("none"), refState: row("left")("state")} }).group("origType", "origPurpose", "refState").count()`,
	},
	{
		name:  "group_two_keys_after_map_object",
		entry: "41",
		expr:  `r.db("restored").table("transactions").getAll("ACCOUNT_REFILL",{index:"type"}).filter({purpose:"refill",state:"transactions/state/completed"}).map(function(t){return {pt: t("paymentType").default("NONE"), sm: t("serviceMethod").default("NONE")}}).group("pt","sm").count()`,
	},
	{
		name:  "group_two_keys_after_map_defaults",
		entry: "59",
		expr:  `r.table("orders").between(1781827200000, r.maxval, {index: "createdAt"}).map(function(o){ return {ch: o("channel").default("none"), pt: o("product")("productType").default(o("product")("category").default("none")), cat: o("category").default("none")} }).group("ch","pt").count()`,
	},
	{
		name:  "group_three_keys_after_map",
		entry: "61",
		expr:  `r.table("orders").between(1781827200000, r.maxval, {index: "createdAt"}).map(function(o){ return {pt: o("product")("productType").default("none"), hasQty: o("quantity").default(null).ne(null), hasAmount: o("amount").default(null).ne(null)} }).group("pt","hasQty","hasAmount").count()`,
	},
	{
		name:  "group_three_keys_after_map_object",
		entry: "78",
		expr:  `r.table("orders").between(1782820375000, r.maxval, {index: "createdAt"}).filter(function(o){ return o("quantity").default(1).gt(1) }).map(function(o){ return {pt: o("product")("productType").default("?"), pr: o("product")("priceType").default("?"), ch: o("channel").default("?")} }).group("pt", "pr", "ch").count()`,
	},
	{
		name:  "group_two_keys_after_merge",
		entry: "36",
		expr: `r.db("restored").table("users").getAll("EaNotLikeUs",{index:"nick"}).nth(0)("id").do(function(uid){
  return r.db("restored").table("accounts").getAll(uid,{index:"userId"})("id").coerceTo("array").do(function(accIds){
    return r.db("restored").table("transactions").getAll(r.args(accIds),{index:"accountBId"})
      .filter({type:"PRODUCT_PURCHASE"})
      .merge(function(t){ return { before22: t("date").lt(1779408000000) } })
      .group("before22", "commissionRate").count()
  })
})`,
	},
	{
		name:  "group_arrow_identity",
		entry: "8",
		expr:  `r.table("orders").filter((o)=>o("couponDiscountAmountUSD").default(null).ne(null)).map((o)=>o("couponDiscountAmountUSD").typeOf()).group((x)=>x).count().ungroup()`,
	},
	{
		name:  "group_arrow_identity_auto_complete_date",
		entry: "14",
		expr:  `r.table("orders").limit(2000).map((o) => o("autoCompleteDate").default(null).typeOf()).group((t) => t).count().ungroup()`,
	},
	{
		name:  "group_arrow_identity_tip_until_time",
		entry: "15",
		expr:  `r.table("orders").limit(2000).map((o) => o("tipUntilTime").default(null).typeOf()).group((t) => t).count().ungroup()`,
	},
	{
		name:  "group_arrow_identity_created_at",
		entry: "16",
		expr:  `r.table("orders").limit(2000).map((o) => o("createdAt").default(null).typeOf()).group((t) => t).count().ungroup()`,
	},
	{
		name:  "group_arrow_identity_accepted_at",
		entry: "17",
		expr:  `r.table("orders").limit(2000).map((o) => o("acceptedAt").default(null).typeOf()).group((t) => t).count().ungroup()`,
	},
	{
		name:  "group_arrow_identity_updated_at",
		entry: "18",
		expr:  `r.table("orders").limit(2000).map((o) => o("updatedAt").default(null).typeOf()).group((t) => t).count().ungroup()`,
	},
	{
		name:  "group_arrow_identity_confirmed_at",
		entry: "19",
		expr:  `r.table("orders").limit(2000).map((o) => o("confirmedAt").default(null).typeOf()).group((t) => t).count().ungroup()`,
	},
	{
		name:  "group_arrow_returning_array",
		entry: "28",
		expr:  `r.table("transactions").filter({type: "ACCOUNT_REFILL", state: "transactions/state/completed"}).filter(t => t("date").gt(1767225600000)).group(t => [t("paymentType").default(null), t("serviceMethod").default(null)]).count()`,
	},
	{
		name:  "group_arrow_array_with_defaults",
		entry: "49",
		expr:  `r.table("transactions").filter({type:"ACCOUNT_REFILL",state:"transactions/state/completed"}).group((t) => [t("paymentType").default("<missing>"),t("serviceMethod").default("<missing>")]).count().ungroup().orderBy(r.desc("reduction"))`,
	},
	{
		name:  "group_arrow_single_default",
		entry: "50",
		expr:  `r.table("transactions").filter({type:"ACCOUNT_REFILL",state:"transactions/state/completed"}).group(t => t("paymentType").default("<missing>")).count().ungroup().orderBy(r.desc("reduction"))`,
	},
	{
		name:  "group_arrow_after_arrow_filter",
		entry: "52",
		expr:  `r.table("transactions").filter(t => t("type").eq("ACCOUNT_REFILL").and(t("state").eq("transactions/state/completed")).and(t("purpose").eq("refill").or(t("purpose").eq("purchase"))).and(t("date").ge(1750204800000))).group(t => t("paymentType").default("<none>")).count()`,
	},
	{
		name:  "group_arrow_distinct_param_name",
		entry: "53",
		expr:  `r.table("transactions").filter(t => t("type").eq("ACCOUNT_REFILL").and(t("state").eq("transactions/state/completed")).and(t("purpose").eq("refill").or(t("purpose").eq("purchase"))).and(t("date").ge(1750204800000)).and(t("paymentType").eq("yuno"))).group(t2 => t2("serviceMethod").default("<none>")).count().ungroup().orderBy(r.desc("reduction"))`,
	},
	{
		name:  "group_arrow_slice",
		entry: "81",
		expr:  `r.db("restored").table("products").group(p => p("id").slice(0,1)).count()`,
	},
	{
		name:  "group_function_single_key",
		entry: "38",
		expr:  `r.db("restored").table("commissions").group(function(c){ return c("type") }).count()`,
	},
	{
		name:  "group_function_array_two_keys",
		entry: "39",
		expr:  `r.db("restored").table("commissions").group(function(c){ return [c("type"), c("primary")] }).count()`,
	},
	{
		name:  "group_function_identity",
		entry: "42",
		expr:  `r.db("restored").table("transactions").getAll("ACCOUNT_REFILL",{index:"type"}).filter({purpose:"refill",state:"transactions/state/completed"}).map(function(t){return t("currencyIn").default("NONE")}).group(function(c){return c}).count()`,
	},
	{
		name:  "group_function_array_three_keys",
		entry: "22",
		expr:  `r.db("restored").table("products").filter({published:true}).group(function(p){return [p("game"),p("category"),p("locale")]}).count().ungroup().orderBy(r.desc("reduction")).limit(8)`,
	},
	{
		name:  "group_function_array_inside_do",
		entry: "46",
		expr:  `r.db("restored").table("transactions").pluck("type","purpose","state").limit(2000).coerceTo("array").do(function(a){return a.group(function(t){return [t("type").default("?"),t("purpose").default("?"),t("state").default("?")]}).count()})`,
	},
	{
		name:  "group_function_after_function_filter",
		entry: "51",
		expr:  `r.table("transactions").filter(function(t){return t("type").eq("ACCOUNT_REFILL").and(t("state").eq("transactions/state/completed")).and(t("purpose").eq("refill").or(t("purpose").eq("purchase"))).and(t("date").ge(1750204800000))}).group(function(t){return t("paymentType").default("<none>")}).count()`,
	},
	{
		name:  "group_function_order_group_id",
		entry: "60",
		expr:  `r.table("orders").between(1781827200000, r.maxval, {index: "createdAt"}).group(function(o){ return o("orderGroupId").default("none") }).count().ungroup().filter(function(g){ return g("reduction").gt(1) }).count()`,
	},
	{
		name:  "group_function_array_orders",
		entry: "64",
		expr:  `r.db("restored").table("orders").limit(3000).group(function(o){ return [o("channel").default("?"), o("productType").default("?")] }).count()`,
	},
	{
		name:  "group_function_array_after_order_by_desc_index",
		entry: "72",
		expr:  `r.db("restored").table("orders").orderBy({index:r.desc("createdAt")}).limit(20000).group(function(o){return [o("channel").default("NO_CHANNEL"), o("product")("productType").default("NO_PT")]}).count()`,
	},
	{
		name:  "group_function_product_type",
		entry: "76",
		expr:  `r.table("orders").between(r.now().sub(7776000), r.maxval, {index: "createdAt"}).filter(function(o){ return o("quantity").default(1).gt(1) }).group(function(o){ return o("product")("productType").default("none") }).count()`,
	},
	{
		name:  "group_function_array_after_pluck",
		entry: "77",
		expr:  `r.table("orders").between(1782820364000, r.maxval, {index: "createdAt"}).filter(function(o){ return o("quantity").default(1).gt(1) }).pluck("channel", "quantity", {"product": ["priceType", "productType"]}).group(function(o){ return [o("product")("productType").default("?"), o("product")("priceType").default("?"), o("channel").default("?")] }).count()`,
	},
	{
		name:  "group_function_and_string_key",
		entry: "35",
		expr: `r.db("restored").table("users").getAll("EaNotLikeUs",{index:"nick"}).nth(0)("id").do(function(uid){
  return r.db("restored").table("accounts").getAll(uid,{index:"userId"})("id").coerceTo("array").do(function(accIds){
    return r.db("restored").table("transactions").getAll(r.args(accIds),{index:"accountBId"})
      .filter({type:"PRODUCT_PURCHASE"})
      .group(function(t){ return t("date").lt(1779408000000) }, "commissionRate").count()
  })
})`,
	},
	{
		name:  "group_zero_param_function",
		entry: "65",
		expr:  `r.db("restored").table("orders").filter({channel:"shop"}).limit(300).group(function(){return true}).map(function(o){ return o("product")("productType").default("ABSENT") }).distinct()`,
	},
	{
		name:  "group_row_has_fields",
		entry: "20",
		expr:  `r.table("products").group(r.row.hasFields("deleted")).count().ungroup()`,
	},
	{
		name:  "group_row_default_payment_type",
		entry: "43",
		expr:  `r.db("restored").table("transactions").getAll("ACCOUNT_REFILL",{index:"type"}).filter({purpose:"refill",state:"transactions/state/completed"}).group(r.row("paymentType").default("NONE")).count()`,
	},
	{
		name:  "group_row_default_service_method",
		entry: "44",
		expr:  `r.db("restored").table("transactions").getAll("ACCOUNT_REFILL",{index:"type"}).filter({purpose:"refill",state:"transactions/state/completed"}).group(r.row("serviceMethod").default("NONE")).count()`,
	},
	{
		name:  "group_row_default_currency_in",
		entry: "45",
		expr:  `r.db("restored").table("transactions").getAll("ACCOUNT_REFILL",{index:"type"}).filter({purpose:"refill",state:"transactions/state/completed"}).group(r.row("currencyIn").default("NONE")).count()`,
	},
	{
		name:  "group_row_slice",
		entry: "80",
		expr:  `r.db("restored").table("products").group(r.row("id").slice(0,1)).count()`,
	},
	{
		name:  "group_array_of_row_keys",
		entry: "23",
		expr:  `r.db("restored").table("products").filter({published:true}).group([r.row("game"),r.row("category"),r.row("locale")]).count().ungroup().orderBy(r.desc("reduction")).limit(8)`,
	},
	// root cause 2: min/max/sum/avg beyond one string literal
	{
		name:  "min_no_args_after_map",
		entry: "29",
		expr:  `r.table("transactions").filter({type: "ACCOUNT_REFILL", state: "transactions/state/completed"}).map(function(t){ return t("date") }).min()`,
	},
	{
		name:  "max_no_args_after_map",
		entry: "30",
		expr:  `r.table("transactions").filter({type: "ACCOUNT_REFILL", state: "transactions/state/completed"}).map(function(t){ return t("date") }).max()`,
	},
	{
		name:  "min_no_args_after_map_with_purpose",
		entry: "31",
		expr:  `r.table("transactions").filter({type: "ACCOUNT_REFILL", state: "transactions/state/completed", purpose: "refill"}).map(function(t){ return t("date") }).min()`,
	},
	{
		name:  "max_no_args_after_map_with_purpose",
		entry: "32",
		expr:  `r.table("transactions").filter({type: "ACCOUNT_REFILL", state: "transactions/state/completed", purpose: "refill"}).map(function(t){ return t("date") }).max()`,
	},
	{
		name:  "min_no_args_after_map_sub",
		entry: "47",
		expr:  `r.db("restored").table("transactions").filter({type:"ACCOUNT_REFILL", purpose:"refill", state:"transactions/state/completed"}).map(function(t){ return t("amountIn").sub(t("commissionAmount").default(0)) }).min()`,
	},
	{
		name:  "min_no_args_after_function_filter",
		entry: "48",
		expr:  `r.db("restored").table("transactions").filter(function(t){ return t("type").eq("ACCOUNT_REFILL").and(t("purpose").default("").eq("refill")).and(t("state").eq("transactions/state/completed")) }).map(function(t){ return t("amountIn").default(0).sub(t("commissionAmount").default(0)) }).min()`,
	},
	{
		name:  "max_index_optargs_then_pluck",
		entry: "7",
		expr:  `r.db("restored").table("orders").max({index: "confirmedAt"}).pluck("id", "confirmedAt")`,
	},
	{
		name:  "min_index_optargs_then_bracket",
		entry: "62",
		expr:  `r.table("orders").min({index: "createdAt"})("createdAt")`,
	},
	{
		name:  "max_index_optargs_then_bracket",
		entry: "67, 68",
		expr:  `r.table("orders").max({index:"createdAt"})("createdAt")`,
	},
	{
		name:  "min_index_optargs_then_bracket_orders",
		entry: "69",
		expr:  `r.table("orders").min({index:"createdAt"})("createdAt")`,
	},
	{
		name:  "min_and_max_index_optargs_inside_do",
		entry: "66",
		expr:  `r.table("orders").between(r.minval, r.maxval, {index:"createdAt"}).do(function(x){return {min: r.table("orders").min({index:"createdAt"})("createdAt"), max: r.table("orders").max({index:"createdAt"})("createdAt")}})`,
	},
	{
		name:  "min_row_nested_bracket_after_get_all",
		entry: "25",
		expr:  `r.table('products').getAll(['wow','gold','us',true],{index:'game_category_locale_published'}).min(r.row('prices')('USD')).pluck('id','specialSort',{prices:'USD'})`,
	},
	{
		name:  "min_row_nested_bracket_after_filter",
		entry: "26",
		expr:  `r.table('products').filter({game:'wow',category:'gold',locale:'us',published:true}).min(r.row('prices')('USD')).pluck('id','specialSort',{prices:'USD'})`,
	},
	{
		name:  "group_then_sum_arrow_nested_field",
		entry: "55",
		expr:  `r.table("accounts").filter(t => t("balance")("amount").gt(0)).group("currency").sum(x => x("balance")("amount")).ungroup().orderBy(r.desc("reduction"))`,
	},
	// root cause 3: infix arithmetic
	{
		name:  "arithmetic_in_between_bounds",
		entry: "6",
		expr:  `r.expr({thirtyDays: r.db("restored").table("orders").between(r.now().sub(60*60*24*30).toEpochTime().mul(1000), r.now().toEpochTime().mul(1000), {index:"confirmedAt"}).filter({state: "state/COMPLETED"}).count(), ninetyDays: r.db("restored").table("orders").between(r.now().sub(60*60*24*90).toEpochTime().mul(1000), r.now().toEpochTime().mul(1000), {index:"confirmedAt"}).filter({state: "state/COMPLETED"}).count(), allTime: r.db("restored").table("orders").between(r.iso8601("2024-08-24T00:00:00Z").toEpochTime().mul(1000), r.now().toEpochTime().mul(1000), {index:"confirmedAt"}).filter({state: "state/COMPLETED"}).count()})`,
	},
	{
		name:  "arithmetic_upper_bound_compound_index_a",
		entry: "56",
		expr:  `r.db("restored").table("transactions").between(["77dd8361-83b0-4bb4-937f-fc7548fc3dda", 1779222884700], ["77dd8361-83b0-4bb4-937f-fc7548fc3dda", 1779222884700+1], {index:"compound_accountId_date", rightBound:"open"}).pluck("id").coerceTo("array")`,
	},
	{
		name:  "arithmetic_upper_bound_compound_index_b",
		entry: "57",
		expr:  `r.db("restored").table("transactions").between(["314f0182-f496-4c4f-a1b3-945ff5b1b7e4", 1779222884700], ["314f0182-f496-4c4f-a1b3-945ff5b1b7e4", 1779222884700+1], {index:"compound_accountId_date", rightBound:"open"}).pluck("id").coerceTo("array")`,
	},
	{
		name:  "arithmetic_inside_now_sub",
		entry: "75",
		expr:  `r.table("orders").between(r.now().sub(60*24*3600), r.maxval, {index: "createdAt"}).filter(function(o){ return o("quantity").default(1).gt(1).and(o("product")("productType").default("").eq("service")) }).limit(8).pluck("game", "channel", "quantity", "priceUSD", "createdAt", {"product": ["priceType", "productType"]})`,
	},
	// root cause 4: top-level r.tableList()
	{
		name:  "table_list_top_level",
		entry: "2, 3, 4",
		expr:  `r.tableList()`,
	},
	// root cause 5: var bindings in function bodies
	{
		name:  "function_local_binding_in_concat_map",
		entry: "1",
		expr: `
r.db("restored").table("routes")
  .getAll("/games/wow/coaching", {index: "url.en"})
  .concatMap(function(route){
    var sw = route("pageConfiguration").default([])
      .filter(function(p){ return p("type").default("").eq("routeSwitcher") })
      .nth(0).default(null);
    return r.branch(
      sw.eq(null),
      [],
      sw("data").default([]).map(function(id){
        return { parentId: route("id"), parentUrl: route("url")("en"), linkedId: id }
      })
    );
  })
  .merge(function(x){
    return r.db("restored").table("routes").get(x("linkedId")).do(function(lr){
      return r.branch(
        lr.eq(null),
        { linkedUrl: null, hasQuickLinkTitle: false, missingRoute: true },
        {
          linkedUrl: lr("url")("en").default(null),
          missingRoute: false,
          hasQuickLinkTitle: lr("pageConfiguration").default([])
            .filter(function(p){ return p("type").default("").eq("quickLinkTitle") })
            .count().gt(0)
        }
      );
    });
  })
`,
	},
	{
		name:  "function_local_regex_binding",
		entry: "79",
		expr:  `r.db("restored").table("products").filter(function(p) { var re = "(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$"; return p("id").default("").coerceTo("string").match(re).eq(null).or(p("userId").default("").coerceTo("string").match(re).eq(null)).or(p("routeId").default("").coerceTo("string").match(re).eq(null)); }).count()`,
	},
	// root cause 6: field selectors with an expression or an array
	{
		name:  "has_fields_with_lambda_param",
		entry: "10",
		expr:  `r.expr(["files","technicalDescriptions","contractors","arbitrageTransaction","timer","options"]).map(function(f){return [f, r.table("orders").filter(function(o){return o.hasFields(f)}).count()]})`,
	},
	{
		name:  "has_fields_array_selector",
		entry: "40",
		expr:  `r.db("restored").table("transactions").getAll("ACCOUNT_REFILL",{index:"type"}).filter({purpose:"refill",state:"transactions/state/completed"}).filter(function(t){return t.hasFields(["meta","manualCompletion"])}).map(function(t){return {hasComplete: t.hasFields("completeDate"), pt: t("paymentType").default("NONE")}}).group("hasComplete","pt").count()`,
	},
	// root cause 7: .branch() chain form
	{
		name:  "branch_chain_in_concat_map",
		entry: "70",
		expr:  `r.db("restored").table("orders").filter({game:"tips"}).hasFields("orderGroupId").limit(20).concatMap(function(o){return r.db("restored").table("orders").getAll(o("orderGroupId"),{index:"orderGroupId"}).filter(function(x){return x("game").ne("tips")}).count().gt(0).branch([o("orderGroupId")],[])})`,
	},
	{
		name:  "branch_chain_in_concat_map_count",
		entry: "71",
		expr:  `r.db("restored").table("orders").filter({game:"tips"}).hasFields("orderGroupId").limit(200).concatMap(function(o){return r.db("restored").table("orders").getAll(o("orderGroupId"),{index:"orderGroupId"}).filter(function(x){return x("game").ne("tips")}).count().gt(0).branch([o("id")],[])}).count()`,
	},
	// root cause 8: bracket notation and getField with an expression
	{
		name:  "bracket_with_lambda_param",
		entry: "12",
		expr:  `r.db("restored").table("products").filter(function(p){ return r.expr(["specialSort","specialSortAllGameProducts","promotionLevel"]).map(function(f){ return p(f).default(null).do(function(v){ return v.typeOf().eq("NUMBER").and(v.ne(v.floor())); }); }).contains(true); }).count()`,
	},
	{
		name:  "get_field_with_lambda_param",
		entry: "13",
		expr:  `r.db("restored").table("products").filter(function(p){ return r.expr(["specialSort","specialSortAllGameProducts","promotionLevel"]).map(function(f){ return p.getField(f).default(null).do(function(v){ return v.typeOf().eq("NUMBER").and(v.ne(v.floor())); }); }).contains(true); }).count()`,
	},
	// root cause 9: term-valued optargs
	{
		name:  "between_and_order_by_desc_index_optargs",
		entry: "27",
		expr:  `r.table("products").between(["411d9a74-1d18-4039-8e97-f59abec40ed3", true, r.minval, r.minval], ["411d9a74-1d18-4039-8e97-f59abec40ed3", true, r.maxval, r.maxval], {index: r.desc("routeId_published_specialSort_prices.USD")}).orderBy({index: r.desc("routeId_published_specialSort_prices.USD")}).limit(2).map(p => ({id: p("id"), specialSort: p("specialSort").default(null), priceUSD: p("prices")("USD").default(null)}))`,
	},
	// root cause 10: arrow lambda with a block body
	{
		name:  "arrow_block_body_group_report",
		entry: "54",
		expr:  `r.table("accounts").filter(t => t("balance")("amount").gt(0)).group("currency").ungroup().map(g => {return {currency:g("group"), accounts:g("reduction").count(), total:g("reduction").sum(x=>x("balance")("amount"))}}).orderBy(r.desc("accounts"))`,
	},
	// root cause 11: slice() with a single argument
	{
		name:  "slice_single_negative_bound",
		entry: "9",
		expr:  `r.db("restored").table("transactions").orderBy({index: "date"}).slice(-2).pluck("id","paymentType","paymentGateway","serviceMethod","meta")`,
	},
	// root cause 12: r.desc() with an expression instead of a string literal
	{
		name:  "desc_arrow_lambda_balance_diff",
		entry: "2026-08-14",
		expr:  `r.db("restored").table("tmp_edge_check").filter(acc => acc("balance")("express").default(0).gt(0)).orderBy(r.desc(acc => acc("balance")("express").sub(acc("balance")("amount")))).pluck("id")`,
	},
	{
		name:  "desc_arrow_lambda_balance_diff_with_defaults",
		entry: "2026-08-14",
		expr:  `r.db("restored").table("tmp_edge_check").filter(acc => acc("balance")("express").default(0).gt(0)).orderBy(r.desc(acc => acc("balance")("express").default(0).sub(acc("balance")("amount").default(0)))).pluck("id")`,
	},
}

// TestParse_ProductionLog_Accepted replays the recorded expressions behind every
// root cause of the parser error log and requires each one to parse.
func TestParse_ProductionLog_Accepted(t *testing.T) {
	t.Parallel()
	for _, tc := range productionLogAccepted {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if _, err := Parse(tc.expr); err != nil {
				t.Fatalf("log entry %s: Parse: %v", tc.entry, err)
			}
		})
	}
}

// productionLogRejected holds the recorded expressions that stay unsupported by
// design, together with the substrings their error message must offer instead.
var productionLogRejected = []struct {
	name     string
	entry    string
	expr     string
	wantMsgs []string
}{
	{
		name:     "new_date_in_compound_between_bound",
		entry:    "73",
		expr:     `r.db("restored").table("transactions").between(["ACCOUNT_CURRENCY_CHANGE", new Date("2026-06-19T07:40:13.981Z").getTime()], ["ACCOUNT_CURRENCY_CHANGE", r.maxval], {index: "compound_type_date"}).filter(r.row.hasFields("commissionAmountUsd")).orderBy("date").limit(1).pluck("id", "date", "state", "commissionAmountUsd")`,
		wantMsgs: []string{"new Date()", "r.iso8601", "r.epochTime"},
	},
	{
		name:     "new_date_in_between_bound_count",
		entry:    "74",
		expr:     `r.db("restored").table("transactions").between(["ACCOUNT_CURRENCY_CHANGE", new Date("2026-06-19T07:40:13.981Z").getTime()], ["ACCOUNT_CURRENCY_CHANGE", r.maxval], {index: "compound_type_date"}).count()`,
		wantMsgs: []string{"new Date()", "r.iso8601", "r.epochTime"},
	},
	{
		name:     "new_date_inside_expr",
		entry:    "83",
		expr:     `r.expr(new Date("2026-01-01"))`,
		wantMsgs: []string{"new Date()", "r.iso8601", "r.epochTime"},
	},
	{
		name:     "table_without_r_prefix_probe",
		entry:    "58",
		expr:     `table("probe").count()`,
		wantMsgs: []string{"unknown identifier \"table\"", "r.table(...)"},
	},
	{
		name:     "table_without_r_prefix",
		entry:    "82",
		expr:     `table("x").count()`,
		wantMsgs: []string{"unknown identifier \"table\"", "r.table(...)"},
	},
	{
		name:     "two_statements_separated_by_semicolon",
		entry:    "5",
		expr:     `r.table("interfaces").filter({normal: "wow-classic_server"}).pluck("id","normal","type"); r.table("interfaces").filter({normal: "wow-classic_server"}).nth(0)("options").count()`,
		wantMsgs: []string{"multiple statements", "one query at a time", "--file", "---"},
	},
	{
		name:     "two_counts_separated_by_semicolon",
		entry:    "84",
		expr:     `r.table("x").count(); r.table("y").count()`,
		wantMsgs: []string{"multiple statements", "one query at a time", "--file", "---"},
	},
	// the rest of this chain -- getAll/filter/group/map/reduce/ungroup with nested
	// r.branch and function bodies -- parses; .abs() is its only blocker, and ReQL has
	// no ABS term to add, so the hint names the r.branch rewrite instead
	{
		name:     "abs_in_vat_reconciliation_chain",
		entry:    "2026-08-17",
		expr:     `r.db("restored").table("transactions")  .getAll("VAT_DEDUCTION","ACCOUNT_REFILL",{index:"type"})  .filter(function(t){return t("state").eq("transactions/state/completed")    .and(t("type").eq("VAT_DEDUCTION").or(t("vatAmount").default(0).gt(0)))    .and(t.hasFields("orderGroupId"))})  .group("orderGroupId")  .map(function(t){return {    vdN:  r.branch(t("type").eq("VAT_DEDUCTION"),1,0),    vdSum:r.branch(t("type").eq("VAT_DEDUCTION"),t("amountIn").default(0),0),    vdCur:r.branch(t("type").eq("VAT_DEDUCTION"),t("currencyIn").default(""),""),    lgN:  r.branch(t("type").eq("ACCOUNT_REFILL"),1,0),    lgSum:r.branch(t("type").eq("ACCOUNT_REFILL"),t("vatAmount").default(0),0),    lgCur:r.branch(t("type").eq("ACCOUNT_REFILL"),t("currencyIn").default(""),"")}})  .reduce(function(a,b){return {    vdN:a("vdN").add(b("vdN")), vdSum:a("vdSum").add(b("vdSum")),    vdCur:r.branch(a("vdCur").eq(""),b("vdCur"),a("vdCur")),    lgN:a("lgN").add(b("lgN")), lgSum:a("lgSum").add(b("lgSum")),    lgCur:r.branch(a("lgCur").eq(""),b("lgCur"),a("lgCur"))}})  .ungroup()  .map(function(g){return g("reduction").merge({cls:    r.branch(g("reduction")("vdN").eq(0), "legacy_only",    r.branch(g("reduction")("vdN").gt(1), "multi_vd",    r.branch(g("reduction")("lgN").eq(0), "vd_only",    r.branch(g("reduction")("lgN").gt(1), "vd_plus_multi_legacy",    r.branch(g("reduction")("vdCur").eq(g("reduction")("lgCur"))      .and(g("reduction")("vdSum").sub(g("reduction")("lgSum")).abs().lt(0.005)),      "agree", "conflict")))))})})  .group("cls").count() `,
		wantMsgs: []string{".abs() is not a ReQL term", "r.branch(x.lt(0), x.mul(-1), x)"},
	},
}

// TestParse_ProductionLog_Rejected pins the actionable hints for the recorded
// expressions that ReQL cannot express, so each error names the supported form.
func TestParse_ProductionLog_Rejected(t *testing.T) {
	t.Parallel()
	for _, tc := range productionLogRejected {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := Parse(tc.expr)
			if err == nil {
				t.Fatalf("log entry %s: expected error, got nil", tc.entry)
			}
			for _, want := range tc.wantMsgs {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("log entry %s: error %q does not contain %q", tc.entry, err.Error(), want)
				}
			}
			if !strings.Contains(err.Error(), "position") {
				t.Errorf("log entry %s: error %q does not include a byte position", tc.entry, err.Error())
			}
		})
	}
}

// TestParse_ProductionLog_RowInsideLambda keeps r.row rejected inside a lambda body,
// where the enclosing parameter is the unambiguous reference. The recorded query was
// re-run successfully with the parameter form, which the accepted table covers.
func TestParse_ProductionLog_RowInsideLambda(t *testing.T) {
	t.Parallel()
	_, err := Parse(`r.db("restored").table("users").getAll("EaNotLikeUs",{index:"nick"}).nth(0)("id").do(function(uid){
  return r.db("restored").table("accounts").getAll(uid,{index:"userId"})("id").coerceTo("array").do(function(accIds){
    return r.db("restored").table("transactions").getAll(r.args(accIds),{index:"accountBId"})
      .filter({type:"PRODUCT_PURCHASE"})
      .group(r.row("date").lt(1779408000000), "commissionRate").count()
  })
})`)
	if err == nil {
		t.Fatal("log entry 34: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "r.row inside") {
		t.Errorf("log entry 34: error %q does not explain the r.row scope rule", err.Error())
	}
}
