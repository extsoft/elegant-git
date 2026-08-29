FROM ruby:3.4-slim

RUN apt-get update \
  && apt-get install -y --no-install-recommends build-essential git \
  && rm -rf /var/lib/apt/lists/*

WORKDIR /srv/docs

COPY Gemfile Gemfile.lock* ./
RUN bundle install

EXPOSE 4000

# Livereload's document.write inject is unreliable through Docker port maps
# and often yields a blank page; --watch alone is enough for local preview.
CMD ["bundle", "exec", "jekyll", "serve", "--host", "0.0.0.0", "--watch", "--force_polling"]
