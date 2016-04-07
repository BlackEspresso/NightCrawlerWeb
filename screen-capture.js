var system = require('system');
var url = system.args[1];
var fileName = system.args[2];
var format = system.args[3];
var size = system.args[4]

var page = require('webpage').create();
// split 1920x1080
var sizeX = size.split('x')[0];
var sizeY = size.split('x')[1];
page.viewportSize = { width: sizeX, height: sizeY };
page.resourceTimeout = 20000;
page.open(url, function() {
	window.setTimeout(function () {
            page.render(fileName,{'format':format,quality:'75'});
  			phantom.exit();
    }, 1000)

});
