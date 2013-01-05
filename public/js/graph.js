function drawCommitChart(color) {

  // Date formatter function
  var parse = d3.time.format("%m/%d/%y").parse;

  // Chart dimensions
  var m = [20, 80, 20, 80]; // Margins
  var w = 960 - m[1] - m[3];
  var h = 150 - m[0] - m[2];

  // Scales. Nice functions which auto resize things.
  // Also defines the ranges for the graph (top and bottom numbers)
  var x = d3.time.scale().range([0, w]);
  var y = d3.scale.linear().range([h, 0]);

  var xAxis = d3.svg.axis().scale(x).tickSize(-h).tickSubdivide(true);
  var yAxis = d3.svg.axis().scale(y).ticks(4).orient("right");

  // An area generator, for the light fill.
  var area = d3.svg.area()
    .interpolate("basis")
    .x(function(d) { return x(d.x); })
    .y0(h)
    .y1(function(d) { return y(d.y); });

  var line = d3.svg.line()
    .interpolate("basis")
    .x(function(d) { return x(d.x); })
    .y(function(d) { return y(d.y); });

  // The minified js import at the top gives us all of the d3 plugins. FOR FREE!
  d3.csv("/data/commit.csv", function(data) {
    var values = data.map(function(d) {
      return { x: parse(d.date), y: +d.commits };
    });

    // Compute the minimum and maximum date, and the maximum y value.
    x.domain([parse(data[0].date), parse(data[data.length - 1].date)]);
    y.domain([0, d3.max(values, function(d) { return d.y; })]).nice();

    // Add an SVG element with the desired dimensions and margin.
    var svg = d3.select("#commits").append("svg:svg")
      .attr("width", w + m[1] + m[3])
      .attr("height", h + m[0] + m[2])
      .append("svg:g")
      .attr("transform", "translate(" + m[3] + "," + m[0] + ")");

    // TODO(icco): get this to work.
    var barPadding = 1;
    svg.selectAll("rect")
      .data(values)
      .enter()
      .append("rect")
      .attr("fill", color)
      .attr("x", function(d) { return x(d.x); })
      .attr("y", function(d) { return y(d.y); })
      .attr("width", 1)
      .attr("height", function(d) { return h - y(d.y); });

    // Add the clip path.
    svg.append("svg:clipPath")
      .attr("id", "clip")
      .append("svg:rect")
      .attr("width", w)
      .attr("height", h);

    // Add the x-axis.
    svg.append("svg:g")
      .attr("class", "x axis")
      .attr("transform", "translate(0," + h + ")")
      .call(xAxis);

    // Add the y-axis.
    svg.append("svg:g")
      .attr("class", "y axis")
      .attr("transform", "translate(" + w + ",0)")
      .call(yAxis);

    // Add a small label for the name.
    svg.append("svg:text")
      .attr("x", w - 6)
      .attr("y", m[0] - 12)
      .attr("text-anchor", "end")
      .text("commits/day");
  });
}

function drawWeeklyChart(color, year, element) {

  // Date formatter function
  var parse = d3.time.format("%Y-%m-%d").parse;

  // Chart dimensions
  var m = [20, 80, 20, 80]; // Margins
  var w = 960 - m[1] - m[3];
  var h = 150 - m[0] - m[2];

  // Scales. Nice functions which auto resize things.
  // Also defines the ranges for the graph (top and bottom numbers)
  var x = d3.time.scale().range([0, w]);
  var y = d3.scale.linear().range([h, 0]);

  var xAxis = d3.svg.axis().scale(x).orient("bottom");
  var yAxis = d3.svg.axis().scale(y).ticks(5).orient("right");

  var line = d3.svg.line()
    .x(function(d) { return x(d.x); })
    .y(function(d) { return y(d.y); });

  d3.csv("/data/"+year+"/weekly.csv", function(data) {
    var values = data.map(function(d) {
      return { x: parse(d.week), y: +d.commits };
    });

    x.domain(d3.extent(values, function(d) { return d.x; }));
    y.domain(d3.extent(values, function(d) { return d.y; }));

    // Add an SVG element with the desired dimensions and margin.
    var svg = d3.select(element).append("svg:svg")
      .attr("width", w + m[1] + m[3])
      .attr("height", h + m[0] + m[2])
      .append("svg:g")
      .attr("transform", "translate(" + m[3] + "," + m[0] + ")");

    svg.append("path")
      .datum(values)
      .attr("class", "line")
      .attr("stroke", color)
      .attr("d", line);

    // Add the x-axis.
    svg.append("svg:g")
      .attr("class", "x axis")
      .attr("transform", "translate(0," + h + ")")
      .call(xAxis);

    // Add the y-axis.
    svg.append("svg:g")
      .attr("class", "y axis")
      .attr("transform", "translate(" + w + ",0)")
      .call(yAxis);

    // Add a small label for the name.
    svg.append("svg:text")
      .attr("x", w - 6)
      .attr("y", m[0] - 12)
      .attr("text-anchor", "end")
      .text("commits/week for " + year);
  });
}
