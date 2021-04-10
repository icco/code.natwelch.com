function drawRepos() {
  var user = new Gh3.User('icco');
  var userRepos = new Gh3.Repositories(user);

  // All of my repos, and their categories
  var repos = [
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

  // Get repos.
  userRepos.fetch(function () {
    repoObjects = userRepos.getRepositories();
    repoObjects.each(function (repo) {
      console.log(repo);
    });
  }, function() {
    // error
  }, {});
}
