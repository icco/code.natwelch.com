DB = Sequel.connect(ENV["DATABASE_URL"] || "sqlite://db/data.db")

class Commit < Sequel::Model(:commits)
  def validate
    super

    validates_presence [ :user, :repo, :sha ]
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
          p event if event["actor"] == "icco" and event["type"] == "PushEvent"
        end
      end
    rescue Timeout::Error
      puts "The request for a page at #{uri} timed out...skipping."
    rescue OpenURI::HTTPError => e
      puts "The request for a page at #{uri} returned an error. #{e.message}"
    end
  end

  def self.factory user, repo, sha
    c = Commit.new
    c.user = user
    c.repo = repo
    c.sha = sha

    if c.valid?
      c.save
      return c
    else
      c.errors.full_messages.each {|error| puts "ERROR SAVING COMMIT: #{error}" }
      return nil
    end
  end
end
