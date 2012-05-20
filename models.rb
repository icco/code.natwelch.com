DB = Sequel.connect(ENV["DATABASE_URL"] || "sqlite://db/data.db")

class Commit < Sequel::Model(:commits)
  plugin :validation_helpers

  def validate
    super

    validates_presence [ :user, :repo, :sha, :created_on ]
    validates_unique   [ :user, :repo, :sha ]
  end

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
          if event["actor"] == "icco" and event["type"] == "PushEvent"
            puts ""

            # TODO(icco): fix this so we record the user name of the commit
            # owner, not the repo owner.
            user = event["repository"]["owner"]
            repo = event["repository"]["name"]
            event["payload"]["shas"].each do |commit|
              sha = commit[0]
              p self.factory user, repo, sha
            end
          end
        end
      end
    rescue Timeout::Error
      puts ""
      puts "The request for #{uri} timed out...skipping."
    rescue OpenURI::HTTPError => e
      puts ""
      puts "The request for #{uri} returned an error. #{e.message}"
    end
  end

  def self.factory user, repo, sha
    c = Commit.new
    c.user = user
    c.repo = repo
    c.sha = sha

    gh_commit = Octokit.commit("#{c.user}/#{c.repo}", sha)
    c.created_on = DateTime.iso8601(gh_commit.commit.author.date)

    if c.valid?
      c.save
      return c
    else
      c.errors.full_messages.each {|error| puts "ERROR SAVING COMMIT: #{error}" }
      return nil
    end
  end
end
