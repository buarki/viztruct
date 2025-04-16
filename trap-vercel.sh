#!/bin/sh

# This script is to trap vercel in production environment :)
# It just renames static directory to public to allow we do 
# a "static files" deploy
mv static public

