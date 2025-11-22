const fs = require("fs");

const raw = fs.readFileSync("input.json", "utf8");
const data = JSON.parse(raw);
let results = data.results || [];

results = results.map(({ profile, ...rest }) => rest);
fs.writeFileSync("output.json", JSON.stringify(results, null, 2));

console.log("Created output.json with results only.");
