class Commit <  ActiveRecord::Base
  USER = "icco"

  validates :user, :presence => true
  validates :repo, :presence => true
  validates :sha, :presence => true, :uniqueness => {:scope => [:user,:repo]}

  def self.fetchAllForTime day, month, year, hour
    require "open-uri"
    require "zlib"
    require "yajl"

    # Simple error checking.
    return nil if hour < 0 or hour > 23
    return nil if day < 1 or day > 31
    return nil if month < 1 or month > 12

    date = "#{"%4d" % year}-#{"%02d" % month}-#{"%02d" % day}-#{hour}"
    uri = URI.parse "http://data.githubarchive.org/#{date}.json.gz"
    begin
      uri.open do |gz|
        js = Zlib::GzipReader.new(gz).read

        Yajl::Parser.parse(js) do |event|
          if event["actor"] == USER and event["type"] == "PushEvent"

            # TODO(icco): fix this so we record the user name of the commit
            # owner, not the repo owner.
            user = event["repository"]["owner"]
            repo = event["repository"]["name"]
            event["payload"]["shas"].each do |commit|
              sha = commit[0]
              ret = self.factory user, repo, sha
              logger.push "Inserted #{ret}.", :info
            end
          end
        end
      end
    rescue Timeout::Error
      logger.push "The request for #{uri} timed out...skipping.", :warn
    rescue OpenURI::HTTPError => e
      logger.push "The request for #{uri} returned an error. #{e.message}", :warn
    end
  end

  def self.factory user, repo, sha
    c = Commit.new

    # Sleep until we have the ratelimit to do this.
    sleep(1) until Octokit.ratelimit.remaining > 2

    begin
      gh_commit = Octokit.commit("#{user}/#{repo}", sha)

      # This is to prevent counting repos I just forked and didn't do any work
      # in. A few commits will still slip through thought that don't belong to
      # me. I don't know why.
      if gh_commit.author and gh_commit.author.login
        c.user = gh_commit.author.login
      else
        c.user = user
      end

      c.repo = repo
      c.sha = sha

      c.created_on = DateTime.iso8601(gh_commit.commit.author.date)

      if c.valid?
        c.save
        return c
      else
        logger.push("Error Saving Commit #{user}/#{repo}:#{c.sha}: #{c.errors.messages.inspect}", :warn)
        return nil
      end
    rescue Octokit::NotFound
      logger.push("Error Saving Commit #{user}/#{repo}:#{sha}: 404", :warn)
    end
  end
end
