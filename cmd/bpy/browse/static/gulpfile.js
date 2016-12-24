var fs = require('fs');
var path = require('path');
var gulp = require('gulp');

// Build environment checks.
var assert = require('assert');
assert.notEqual(process.env['GOPATH'], undefined, 'GOPATH must be set and contain github.com/buppyio/bpy');

var categoriesPath = path.join(process.env['GOPATH'], './src/github.com/buppyio/bpy/doc/man/');
var partialsPath = './templates/';

// CSS less, autoprefix and minify.
var less = require('gulp-less');
var autoprefixer = require('gulp-autoprefixer');
var cleanCSS = require('gulp-clean-css');
var cssSources = './less/*.less';
var cssOutputsDir = './www/css/';

gulp.task('css', function() {
  return gulp
    .src(cssSources)
    .pipe(less())
    .pipe(autoprefixer())
    .pipe(cleanCSS())
    .pipe(gulp.dest(cssOutputsDir));
});

// Copy library resources.
gulp.task('libs', function() {
  return [
    gulp.src('./node_modules/bootstrap/dist/fonts/*').pipe(gulp.dest('./www/fonts')),
    //gulp.src('./node_modules/bootstrap/dist/js/*').pipe(gulp.dest('./www/js')),
    gulp.src('./node_modules/jquery/dist/*').pipe(gulp.dest('./www/js'))
  ];
});

// Helpers.
gulp.task('default', ['css', 'libs']);

gulp.task('watch', ['default'], function(){
  gulp.watch(cssSources, ['css']);
});
