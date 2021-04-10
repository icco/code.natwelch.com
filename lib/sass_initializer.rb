# frozen_string_literal: true

module SassInitializer
  def self.registered(app)
    # Enables support for SASS template reloading in rack applications.
    # See http://nex-3.com/posts/88-sass-supports-rack for more details.
    # Store SASS files (by default) within 'app/stylesheets'.
    require "rack/sassc"

    app.use Rack::SassC, {
      css_location: "public/css",
      style: :compact,
      template_location: "css"
    }
  end
end
