function drawRepos() {
  // Get GitHub Repositories
  // TODO: if over 100 repos, add pagination.
  var login = 'icco';
  $.getJSON('https://api.github.com/users/' + login + '/repos?type=owner&per_page=100&sort=pushed&callback=?', function(data) {
    var repos = data.data;

     // The repos I want to feature
     var myRepos = [
        'Agent355',
        'CSC484',
        'Javascript_Embed',
        'RainbowDeathSwarm',
        'Resume',
        'bloomFilter',
        'coffee_shop',
        'dotFiles',
        'pseudoweb',
        'thestack',
     ];

     // Split into featured and other
     other_repos = repos.filter(function (x) { return myRepos.indexOf(x.name) < 0 && !x.private; });
     featured_repos = repos.filter(function (x) { return myRepos.indexOf(x.name) >= 0; });

     // Detailed repositories
     $.each(featured_repos, function () {
        hp = '<a href="' + this.homepage + '">#</a>';
        desc = '<small> - ' + this.description + '</small>';
        a = '<a href="' + this.html_url + '">' + this.name + '</a> ';
        a = this.homepage ? a + hp : a;

        $('#repos > ul').append('<li>' + a + '<br />' + desc + '</li>')
     });

     // Everything else.
     $.each(other_repos, function () {
        a = '<a href="' + this.html_url + '">' + this.name + '</a> ';
        $('#other_repos > ul').append('<li>' + a + '</li>');
     });

     $('#other_repos > h3').bind('click', function (ev) {
        $('#other_repos > ul').fadeToggle();
     });
  });
}
